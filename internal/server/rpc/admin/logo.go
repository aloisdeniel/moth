package adminrpc

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"image/png"
	"io"
	"strings"
)

// Logo upload constraints. The size cap matches the proto contract
// (theme.proto); the dimension bounds keep hosted pages and mobile screens
// from decoding wallpaper-sized images.
const (
	maxLogoBytes = 512 * 1024
	minLogoDim   = 8
	maxLogoDim   = 2048
)

// processLogo turns an uploaded image into the bytes stored and served. It
// never stores the upload verbatim: PNGs are decoded and re-encoded (which
// drops every ancillary chunk and any payload hidden in one), SVGs are
// parsed and rebuilt from an element/attribute allowlist. Errors are
// user-facing (invalid argument).
func processLogo(data []byte, contentType string) (clean []byte, ext string, err error) {
	switch contentType {
	case "image/png":
		// Check the declared dimensions from the header alone before
		// png.Decode runs: Decode allocates the whole width*height pixel
		// buffer from the IHDR up front, so a tiny crafted file declaring
		// wallpaper dimensions would otherwise force a multi-GB allocation
		// that the bounds check never gets to see.
		cfg, err := png.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			return nil, "", fmt.Errorf("invalid PNG: %v", err)
		}
		if cfg.Width < minLogoDim || cfg.Height < minLogoDim {
			return nil, "", fmt.Errorf("logo is %dx%d px; at least %dx%d is required", cfg.Width, cfg.Height, minLogoDim, minLogoDim)
		}
		if cfg.Width > maxLogoDim || cfg.Height > maxLogoDim {
			return nil, "", fmt.Errorf("logo is %dx%d px; at most %dx%d is accepted", cfg.Width, cfg.Height, maxLogoDim, maxLogoDim)
		}
		img, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, "", fmt.Errorf("invalid PNG: %v", err)
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return nil, "", fmt.Errorf("re-encode PNG: %v", err)
		}
		return buf.Bytes(), "png", nil
	case "image/svg+xml":
		out, err := sanitizeSVG(data)
		if err != nil {
			return nil, "", fmt.Errorf("invalid SVG: %v", err)
		}
		return out, "svg", nil
	default:
		return nil, "", fmt.Errorf("unsupported content type %q (want image/png or image/svg+xml)", contentType)
	}
}

// svgElements is the allowlist of SVG elements a logo may use. Anything
// else — <script> and <foreignObject> above all — is dropped with its whole
// subtree.
var svgElements = map[string]bool{
	"svg": true, "g": true, "defs": true, "title": true, "desc": true,
	"path": true, "rect": true, "circle": true, "ellipse": true,
	"line": true, "polyline": true, "polygon": true,
	"text": true, "tspan": true, "textPath": true,
	"linearGradient": true, "radialGradient": true, "stop": true,
	"clipPath": true, "mask": true, "pattern": true, "symbol": true,
	"use": true, "style": true,
}

const svgNamespace = "http://www.w3.org/2000/svg"

// sanitizeSVG parses an SVG document and re-serializes only what the
// allowlist admits:
//
//   - elements outside svgElements are dropped subtree and all;
//   - event-handler attributes (on*) are dropped;
//   - href / xlink:href survive only as local fragment references ("#id") —
//     external and data: URLs are dropped;
//   - style attributes and <style> content that reference url(...),
//     @import or expression(...) are dropped;
//   - comments, processing instructions and the doctype are dropped.
//
// A document that does not parse as XML with an <svg> root is rejected.
func sanitizeSVG(data []byte) ([]byte, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	var out bytes.Buffer
	enc := xml.NewEncoder(&out)

	var open []string // kept-element stack, root first
	skip := 0         // depth inside a dropped subtree
	seenRoot := false

	// Character data is buffered until the next element boundary instead of
	// being checked token by token: dropped tokens in between (comments,
	// skipped subtrees) would otherwise split one run of text — `ur<!---->l(`
	// — into chunks that individually pass suspiciousCSS but reassemble in
	// the output.
	var text bytes.Buffer
	flushText := func() error {
		if text.Len() == 0 {
			return nil
		}
		data := append([]byte(nil), text.Bytes()...)
		text.Reset()
		if len(open) > 0 && open[len(open)-1] == "style" && suspiciousCSS(string(data)) {
			return nil
		}
		return enc.EncodeToken(xml.CharData(data))
	}

	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if skip > 0 {
				skip++
				continue
			}
			local := t.Name.Local
			if !seenRoot {
				if local != "svg" {
					return nil, fmt.Errorf("root element is <%s>, not <svg>", local)
				}
				seenRoot = true
			}
			if !svgElements[local] {
				// Dropped subtree: keep buffering, so text split around it
				// is checked as the one run it becomes in the output.
				skip = 1
				continue
			}
			if err := flushText(); err != nil {
				return nil, err
			}
			start := xml.StartElement{Name: xml.Name{Local: local}}
			for _, a := range t.Attr {
				if attr, ok := sanitizeSVGAttr(a); ok {
					start.Attr = append(start.Attr, attr)
				}
			}
			if len(open) == 0 {
				// The decoder expanded namespaces away; pin the one that
				// matters back on the root.
				start.Attr = append(start.Attr,
					xml.Attr{Name: xml.Name{Local: "xmlns"}, Value: svgNamespace})
			}
			open = append(open, local)
			if err := enc.EncodeToken(start); err != nil {
				return nil, err
			}
		case xml.EndElement:
			if skip > 0 {
				skip--
				continue
			}
			if len(open) == 0 {
				return nil, errors.New("unbalanced end element")
			}
			if err := flushText(); err != nil {
				return nil, err
			}
			name := open[len(open)-1]
			open = open[:len(open)-1]
			if err := enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: name}}); err != nil {
				return nil, err
			}
		case xml.CharData:
			if skip > 0 || len(open) == 0 {
				continue
			}
			text.Write(t)
		default:
			// Comments, processing instructions, directives (doctype):
			// nothing a logo needs.
		}
	}
	if !seenRoot {
		return nil, errors.New("no <svg> element")
	}
	if err := enc.Flush(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// sanitizeSVGAttr filters one attribute, returning the (possibly renamed)
// attribute to keep.
func sanitizeSVGAttr(a xml.Attr) (xml.Attr, bool) {
	local := strings.ToLower(a.Name.Local)
	switch {
	case strings.HasPrefix(local, "on"):
		// Event handlers (onload, onclick, ...): script by another door.
		return xml.Attr{}, false
	case local == "xmlns" || a.Name.Space == "xmlns":
		// Namespace declarations are rebuilt on the root.
		return xml.Attr{}, false
	case local == "href":
		// Covers href and xlink:href (the decoder resolves the prefix into
		// Name.Space). Only same-document references stay.
		if !strings.HasPrefix(a.Value, "#") {
			return xml.Attr{}, false
		}
		return xml.Attr{Name: xml.Name{Local: "href"}, Value: a.Value}, true
	case local == "style" && suspiciousCSS(a.Value):
		return xml.Attr{}, false
	}
	return xml.Attr{Name: xml.Name{Local: a.Name.Local}, Value: a.Value}, true
}

// suspiciousCSS reports whether inline CSS tries to load anything: url(...)
// (external resources), @import, or legacy expression(...). A backslash is
// suspicious by itself — CSS identifier escapes (`\75 rl(` is `url(`) can
// spell any banned token past a substring check, and a logo has no
// legitimate use for them.
func suspiciousCSS(css string) bool {
	l := strings.ToLower(css)
	return strings.Contains(l, `\`) ||
		strings.Contains(l, "url(") ||
		strings.Contains(l, "@import") ||
		strings.Contains(l, "expression(")
}
