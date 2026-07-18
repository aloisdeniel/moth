package profile

import (
	"testing"

	"google.golang.org/protobuf/proto"

	projectconfigv1 "github.com/aloisdeniel/moth/gen/moth/projectconfig/v1"
)

func TestEncodeParseRoundTrip(t *testing.T) {
	in := Config{
		Platforms:          []string{PlatformIOS, PlatformWeb},
		GoogleSignIn:       true,
		AppleSignIn:        true,
		SellsSubscriptions: true,
		SendsPushes:        true,
		ChecklistDismissed: true,
	}
	raw, err := Encode(in)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	got, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.Version != SchemaVersion {
		t.Fatalf("version: %d", got.Version)
	}
	if len(got.Platforms) != 2 || got.Platforms[0] != PlatformIOS || got.Platforms[1] != PlatformWeb {
		t.Fatalf("platforms: %v", got.Platforms)
	}
	if !got.GoogleSignIn || !got.AppleSignIn || !got.SellsSubscriptions ||
		!got.SendsPushes || !got.ChecklistDismissed {
		t.Fatalf("flags: %+v", got)
	}
}

func TestFromStored(t *testing.T) {
	// Empty bytes: no profile (a pre-wizard project).
	if _, ok := FromStored(nil); ok {
		t.Fatal("empty bytes read as a profile")
	}
	// A valid document parses.
	raw, err := Encode(Config{Platforms: []string{PlatformAndroid}})
	if err != nil {
		t.Fatal(err)
	}
	c, ok := FromStored(raw)
	if !ok || !c.HasPlatform(PlatformAndroid) {
		t.Fatalf("stored profile: %+v ok=%v", c, ok)
	}
	// A future schema version defensively reads as "no profile".
	future, err := proto.Marshal(&projectconfigv1.StoredProfile{Version: SchemaVersion + 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := FromStored(future); ok {
		t.Fatal("future schema read as a profile")
	}
	if _, err := Parse(future); err == nil {
		t.Fatal("Parse accepted a future schema version")
	}
	// Empty input parses as version 0, which Parse must reject.
	if _, err := Parse(nil); err == nil {
		t.Fatal("Parse accepted empty input")
	}
}

func TestValidate(t *testing.T) {
	valid := Config{Version: SchemaVersion, Platforms: []string{PlatformWeb}}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid profile rejected: %v", err)
	}
	cases := map[string]Config{
		"wrong version":      {Version: 2, Platforms: []string{PlatformWeb}},
		"no platforms":       {Version: SchemaVersion},
		"unknown platform":   {Version: SchemaVersion, Platforms: []string{"flutter"}},
		"duplicate platform": {Version: SchemaVersion, Platforms: []string{PlatformIOS, PlatformIOS}},
	}
	for name, c := range cases {
		if err := c.Validate(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestHasPlatform(t *testing.T) {
	c := Config{Platforms: []string{PlatformIOS, PlatformAndroid}}
	if !c.HasPlatform(PlatformIOS) || !c.HasPlatform(PlatformAndroid) || c.HasPlatform(PlatformWeb) {
		t.Fatalf("HasPlatform: %+v", c.Platforms)
	}
}
