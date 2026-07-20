package events

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestParsePlatform(t *testing.T) {
	cases := []struct {
		raw, want string
	}{
		{"", ""},
		{"   ", ""},
		{"ios", PlatformIOS},
		{"IOS", PlatformIOS},
		{" Android ", PlatformAndroid},
		{"web", PlatformWeb},
		{"macos", PlatformMacOS},
		{"windows", PlatformWindows},
		{"linux", PlatformLinux},
		{"other", PlatformOther},
		{"fuchsia", PlatformOther},
		{"<script>alert(1)</script>", PlatformOther},
		{strings.Repeat("x", 10_000), PlatformOther},
	}
	for _, c := range cases {
		if got := ParsePlatform(c.raw); got != c.want {
			t.Errorf("ParsePlatform(%q) = %q, want %q", c.raw, got, c.want)
		}
	}
}

func TestParseSDKVersion(t *testing.T) {
	cases := []struct {
		raw, want string
	}{
		{"", ""},
		{"   ", ""},
		{"1.2.3", "1.2.3"},
		{" 1.2.3\n", "1.2.3"},
		{"0.1.0-dev.4+2", "0.1.0-dev.4+2"},
		{"moth_flutter 1.0.0", "moth_flutter 1.0.0"},
		{strings.Repeat("1", 32), strings.Repeat("1", 32)},
		{strings.Repeat("1", 33), ""},
		{strings.Repeat("1", 10_000), ""},
		{"1.2.3\x00", ""},
		{"1.2.3é", ""},
		{"a;b", ""},
	}
	for _, c := range cases {
		if got := ParseSDKVersion(c.raw); got != c.want {
			t.Errorf("ParseSDKVersion(%q) = %q, want %q", c.raw, got, c.want)
		}
	}
}

func TestClientInfoFromHeader(t *testing.T) {
	h := http.Header{}
	h.Set(PlatformHeader, "ios")
	h.Set(SDKVersionHeader, "1.0.0")
	if got := ClientInfoFromHeader(h); got != (ClientInfo{Platform: "ios", SDKVersion: "1.0.0"}) {
		t.Fatalf("ClientInfoFromHeader = %+v", got)
	}

	if got := ClientInfoFromHeader(http.Header{}); got != (ClientInfo{}) {
		t.Fatalf("missing headers: got %+v, want zero", got)
	}
}

func TestClientInfoContextRoundTrip(t *testing.T) {
	if got := ClientInfoFromContext(context.Background()); got != (ClientInfo{}) {
		t.Fatalf("empty context: got %+v, want zero", got)
	}

	info := ClientInfo{Platform: PlatformAndroid, SDKVersion: "2.1.0"}
	ctx := WithClientInfo(context.Background(), info)
	if got := ClientInfoFromContext(ctx); got != info {
		t.Fatalf("round trip: got %+v, want %+v", got, info)
	}
}
