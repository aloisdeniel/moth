package theme

import (
	"math"
	"testing"
)

func TestParseColor(t *testing.T) {
	tests := []struct {
		in      string
		want    Color
		wantErr bool
	}{
		{in: "#FFFFFF", want: Color{255, 255, 255}},
		{in: "#ffffff", want: Color{255, 255, 255}},
		{in: "#000000", want: Color{}},
		{in: "#6750A4", want: Color{0x67, 0x50, 0xA4}},
		{in: "#6750a4", want: Color{0x67, 0x50, 0xA4}},
		{in: "", wantErr: true},
		{in: "FFFFFF", wantErr: true},
		{in: "#FFF", wantErr: true},      // shorthand rejected
		{in: "#FFFFFFFF", wantErr: true}, // alpha rejected
		{in: "#GGGGGG", wantErr: true},   // not hex
		{in: "#FFFFF ", wantErr: true},   // trailing junk
		{in: "monokai-purple", wantErr: true},
	}
	for _, tc := range tests {
		got, err := ParseColor(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParseColor(%q): want error, got %v", tc.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseColor(%q): %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseColor(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestHexRoundTrip(t *testing.T) {
	for _, hex := range []string{"#000000", "#FFFFFF", "#6750A4", "#0A0B0C"} {
		c, err := ParseColor(hex)
		if err != nil {
			t.Fatal(err)
		}
		if got := c.Hex(); got != hex {
			t.Errorf("Hex() = %q, want %q", got, hex)
		}
	}
}

func TestContrastRatio(t *testing.T) {
	tests := []struct {
		a, b string
		want float64
	}{
		{"#FFFFFF", "#000000", 21},   // maximum
		{"#000000", "#FFFFFF", 21},   // symmetric
		{"#FFFFFF", "#FFFFFF", 1},    // identical
		{"#777777", "#FFFFFF", 4.48}, // well-known just-below-AA gray
		{"#FF0000", "#FFFFFF", 4.00}, // pure red on white
		{"#6750A4", "#FFFFFF", 6.44}, // default primary on white
	}
	for _, tc := range tests {
		a := mustColor(t, tc.a)
		b := mustColor(t, tc.b)
		got := ContrastRatio(a, b)
		if math.Abs(got-tc.want) > 0.01 {
			t.Errorf("ContrastRatio(%s, %s) = %.4f, want %.2f", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestLuminanceBounds(t *testing.T) {
	if l := (Color{}).Luminance(); l != 0 {
		t.Errorf("black luminance = %v, want 0", l)
	}
	if l := (Color{255, 255, 255}).Luminance(); math.Abs(l-1) > 1e-9 {
		t.Errorf("white luminance = %v, want 1", l)
	}
}

func mustColor(t *testing.T, hex string) Color {
	t.Helper()
	c, err := ParseColor(hex)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
