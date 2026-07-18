// Package push defines the per-project push settings: the enabled switch for
// the push-device registry (milestone 20) and the Web Push VAPID public key
// browser clients subscribe with. Plain, non-secret config — the matching
// VAPID private key stays with the developer's sender and never touches moth.
//
// Like the paywall config it is a small versioned document stored on the
// project row (as a moth.projectconfig.v1.StoredPush protobuf message), but
// deliberately simpler: a full-replacement setting with no revision history.
// It is delivered to clients through the public moth.auth.v1.GetProjectConfig
// response.
package push

import (
	"encoding/base64"
	"fmt"

	"google.golang.org/protobuf/proto"

	projectconfigv1 "github.com/aloisdeniel/moth/gen/moth/projectconfig/v1"
)

// SchemaVersion is stamped on every encoded push-settings document. Parse
// rejects documents from a different version; future schema changes bump it
// and add an explicit upgrade path.
const SchemaVersion = 1

// vapidPublicKeyLen is the byte length of an uncompressed P-256 public key
// point (0x04 || X || Y), which is what the Web Push `applicationServerKey`
// must decode to.
const vapidPublicKeyLen = 65

// Config is one project's push settings.
type Config struct {
	// Version is the schema version of the document (SchemaVersion).
	Version int
	// Enabled is the master switch for the push registry; when false the
	// client-facing moth.push.v1 RPCs refuse registrations.
	Enabled bool
	// WebPushVAPIDPublicKey is the VAPID public key (base64url, uncompressed
	// P-256 point) browser clients subscribe with; empty when the project
	// does not use Web Push.
	WebPushVAPIDPublicKey string
}

// Default returns the settings of a project that never configured push:
// disabled, no VAPID key.
func Default() Config {
	return Config{Version: SchemaVersion}
}

// Encode serializes the config as its canonical storage document (a
// moth.projectconfig.v1.StoredPush protobuf message), stamping the current
// schema version.
func Encode(c Config) ([]byte, error) {
	c.Version = SchemaVersion
	raw, err := proto.Marshal(ToProto(c))
	if err != nil {
		return nil, fmt.Errorf("encode push settings: %w", err)
	}
	return raw, nil
}

// Parse decodes a stored push-settings document
// (moth.projectconfig.v1.StoredPush). It rejects documents from a different
// schema version — including empty input, which callers treat as "push never
// configured" before parsing; it does not validate values (Validate does).
func Parse(raw []byte) (Config, error) {
	var msg projectconfigv1.StoredPush
	if err := proto.Unmarshal(raw, &msg); err != nil {
		return Config{}, fmt.Errorf("parse push settings: %w", err)
	}
	if msg.Version != SchemaVersion {
		return Config{}, fmt.Errorf("parse push settings: unsupported schema version %d (want %d)", msg.Version, SchemaVersion)
	}
	return FromProto(&msg), nil
}

// FromStored returns the settings a project effectively runs with: the parsed
// stored document, or Default() when the project never configured push (empty
// bytes) or — defensively — when the stored document cannot be parsed (a
// newer schema or corruption reads as "disabled", never as an error on the
// hot paths).
func FromStored(raw []byte) Config {
	if len(raw) == 0 {
		return Default()
	}
	c, err := Parse(raw)
	if err != nil {
		return Default()
	}
	return c
}

// ToProto converts the domain config into its storage message.
func ToProto(c Config) *projectconfigv1.StoredPush {
	return &projectconfigv1.StoredPush{
		Version:               int32(c.Version),
		Enabled:               c.Enabled,
		WebpushVapidPublicKey: c.WebPushVAPIDPublicKey,
	}
}

// FromProto converts a storage message into the domain config.
func FromProto(msg *projectconfigv1.StoredPush) Config {
	return Config{
		Version:               int(msg.GetVersion()),
		Enabled:               msg.GetEnabled(),
		WebPushVAPIDPublicKey: msg.GetWebpushVapidPublicKey(),
	}
}

// ValidateVAPIDPublicKey checks a non-empty VAPID public key for shape:
// base64url without padding, decoding to an uncompressed P-256 public point
// (65 bytes starting 0x04) — what the browser's `applicationServerKey`
// requires. It is the single server-side rule; Config.Validate and any
// prompt-time validation (the CLI wizard, `moth setup`) call it so a typo is
// caught the same way everywhere.
func ValidateVAPIDPublicKey(key string) error {
	raw, err := base64.RawURLEncoding.DecodeString(key)
	if err != nil {
		return fmt.Errorf("not base64url: %w", err)
	}
	if len(raw) != vapidPublicKeyLen || raw[0] != 0x04 {
		return fmt.Errorf("must decode to a %d-byte uncompressed P-256 point (got %d bytes)", vapidPublicKeyLen, len(raw))
	}
	return nil
}

// Validate checks the config and returns the first violation. The VAPID key
// is validated for shape only (base64url, uncompressed P-256 point) — moth
// never uses it to send, so a key the browser would reject is the only
// mistake worth catching here.
func (c Config) Validate() error {
	if c.Version != SchemaVersion {
		return fmt.Errorf("unsupported schema version %d (want %d)", c.Version, SchemaVersion)
	}
	if c.WebPushVAPIDPublicKey == "" {
		return nil
	}
	if err := ValidateVAPIDPublicKey(c.WebPushVAPIDPublicKey); err != nil {
		return fmt.Errorf("webpushVapidPublicKey: %w", err)
	}
	return nil
}
