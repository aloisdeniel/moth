// Package profile defines the per-project setup profile (milestone 22): the
// creation wizard's answers — platforms, sign-in intent, monetization and
// push intent, plus the checklist-dismissed flag. The profile records what
// the app *intends*, so adaptive surfaces (the setup tab, the derived
// checklist) can tell "doesn't want Apple sign-in" apart from "hasn't
// configured it yet"; it is never a second source of config truth.
//
// Like the push settings it is a small versioned document stored on the
// project row (as a moth.projectconfig.v1.StoredProfile protobuf message):
// a full-replacement setting with no revision history. Unlike every other
// config document it has no default — a project without a stored profile
// simply has none, and every adaptive surface behaves as it did before the
// milestone.
package profile

import (
	"fmt"

	"google.golang.org/protobuf/proto"

	projectconfigv1 "github.com/aloisdeniel/moth/gen/moth/projectconfig/v1"
)

// SchemaVersion is stamped on every encoded profile document. Parse rejects
// documents from a different version; future schema changes bump it and add
// an explicit upgrade path.
const SchemaVersion = 1

// Platforms an app ships on (Config.Platforms values).
const (
	PlatformIOS     = "ios"
	PlatformAndroid = "android"
	PlatformWeb     = "web"
)

// Config is one project's setup profile.
type Config struct {
	// Version is the schema version of the document (SchemaVersion).
	Version int
	// Platforms the app ships on (PlatformIOS/PlatformAndroid/PlatformWeb).
	// Non-empty in every valid profile.
	Platforms []string
	// GoogleSignIn / AppleSignIn record the social sign-in intent;
	// email/password is always on and needs no flag.
	GoogleSignIn bool
	AppleSignIn  bool
	// SellsSubscriptions records the monetization intent (milestones 11/12).
	SellsSubscriptions bool
	// SendsPushes records the push intent (milestone 20).
	SendsPushes bool
	// ChecklistDismissed hides the overview checklist card; it never fakes
	// completeness.
	ChecklistDismissed bool
}

// HasPlatform reports whether the profile includes the platform.
func (c Config) HasPlatform(p string) bool {
	for _, have := range c.Platforms {
		if have == p {
			return true
		}
	}
	return false
}

// Encode serializes the config as its canonical storage document (a
// moth.projectconfig.v1.StoredProfile protobuf message), stamping the
// current schema version.
func Encode(c Config) ([]byte, error) {
	c.Version = SchemaVersion
	raw, err := proto.Marshal(ToProto(c))
	if err != nil {
		return nil, fmt.Errorf("encode profile: %w", err)
	}
	return raw, nil
}

// Parse decodes a stored profile document
// (moth.projectconfig.v1.StoredProfile). It rejects documents from a
// different schema version — including empty input, which callers treat as
// "no profile" before parsing; it does not validate values (Validate does).
func Parse(raw []byte) (Config, error) {
	var msg projectconfigv1.StoredProfile
	if err := proto.Unmarshal(raw, &msg); err != nil {
		return Config{}, fmt.Errorf("parse profile: %w", err)
	}
	if msg.Version != SchemaVersion {
		return Config{}, fmt.Errorf("parse profile: unsupported schema version %d (want %d)", msg.Version, SchemaVersion)
	}
	return FromProto(&msg), nil
}

// FromStored returns the project's profile and whether one exists: the
// parsed stored document, or (zero, false) when the project has no profile
// (empty bytes — created before the wizard) or — defensively — when the
// stored document cannot be parsed (a newer schema or corruption reads as
// "no profile", never as an error: every adaptive surface then behaves as
// pre-milestone).
func FromStored(raw []byte) (Config, bool) {
	if len(raw) == 0 {
		return Config{}, false
	}
	c, err := Parse(raw)
	if err != nil {
		return Config{}, false
	}
	return c, true
}

// ToProto converts the domain config into its storage message.
func ToProto(c Config) *projectconfigv1.StoredProfile {
	msg := &projectconfigv1.StoredProfile{
		Version:            int32(c.Version),
		GoogleSignIn:       c.GoogleSignIn,
		AppleSignIn:        c.AppleSignIn,
		SellsSubscriptions: c.SellsSubscriptions,
		SendsPushes:        c.SendsPushes,
		ChecklistDismissed: c.ChecklistDismissed,
	}
	for _, p := range c.Platforms {
		msg.Platforms = append(msg.Platforms, platformToProto(p))
	}
	return msg
}

// FromProto converts a storage message into the domain config. Unknown
// platform values (a newer schema) are dropped rather than invented.
func FromProto(msg *projectconfigv1.StoredProfile) Config {
	c := Config{
		Version:            int(msg.GetVersion()),
		GoogleSignIn:       msg.GetGoogleSignIn(),
		AppleSignIn:        msg.GetAppleSignIn(),
		SellsSubscriptions: msg.GetSellsSubscriptions(),
		SendsPushes:        msg.GetSendsPushes(),
		ChecklistDismissed: msg.GetChecklistDismissed(),
	}
	for _, p := range msg.GetPlatforms() {
		if name := platformFromProto(p); name != "" {
			c.Platforms = append(c.Platforms, name)
		}
	}
	return c
}

func platformToProto(p string) projectconfigv1.Platform {
	switch p {
	case PlatformIOS:
		return projectconfigv1.Platform_PLATFORM_IOS
	case PlatformAndroid:
		return projectconfigv1.Platform_PLATFORM_ANDROID
	case PlatformWeb:
		return projectconfigv1.Platform_PLATFORM_WEB
	default:
		return projectconfigv1.Platform_PLATFORM_UNSPECIFIED
	}
}

func platformFromProto(p projectconfigv1.Platform) string {
	switch p {
	case projectconfigv1.Platform_PLATFORM_IOS:
		return PlatformIOS
	case projectconfigv1.Platform_PLATFORM_ANDROID:
		return PlatformAndroid
	case projectconfigv1.Platform_PLATFORM_WEB:
		return PlatformWeb
	default:
		return ""
	}
}

// Validate checks the config and returns the first violation: the schema
// version, at least one platform, only known platforms, no duplicates.
func (c Config) Validate() error {
	if c.Version != SchemaVersion {
		return fmt.Errorf("unsupported schema version %d (want %d)", c.Version, SchemaVersion)
	}
	if len(c.Platforms) == 0 {
		return fmt.Errorf("platforms: at least one platform is required")
	}
	seen := map[string]bool{}
	for _, p := range c.Platforms {
		switch p {
		case PlatformIOS, PlatformAndroid, PlatformWeb:
		default:
			return fmt.Errorf("platforms: unknown platform %q", p)
		}
		if seen[p] {
			return fmt.Errorf("platforms: duplicate platform %q", p)
		}
		seen[p] = true
	}
	return nil
}
