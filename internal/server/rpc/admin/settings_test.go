package adminrpc

import (
	"context"
	"encoding/json"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/internal/audit"
	"github.com/aloisdeniel/moth/internal/config"
	"github.com/aloisdeniel/moth/internal/keys"
	"github.com/aloisdeniel/moth/internal/mail"
	"github.com/aloisdeniel/moth/internal/store"
)

// TestSettingsMigratesPlaintextSMTPPassword: an install upgraded from before
// the secrets-at-rest hardening keeps its SMTP password in cleartext inside the
// instance_settings JSON. Building the settings handler must migrate it: the
// password moves into the encrypted instance_secrets row and is stripped from
// the JSON, without waiting for an admin to re-save.
func TestSettingsMigratesPlaintextSMTPPassword(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	master, err := keys.LoadOrCreateMasterKey(t.TempDir(), func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}

	// Simulate a pre-milestone-10 row: the full SMTP config, password included,
	// marshaled into the plaintext settings JSON.
	legacy := config.SMTP{Host: "smtp.example.com", Port: 587, Username: "user", Password: "s3cret", From: "no-reply@example.com"}
	raw, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SetInstanceSetting(ctx, store.InstanceSettingSMTP, string(raw), time.Now()); err != nil {
		t.Fatal(err)
	}

	log := slog.New(slog.DiscardHandler)
	dyn := mail.NewDynamic(mail.Console{Log: log})
	auditor := NewAuditor(audit.New(st, log, time.Now))
	h, err := NewSettingsHandler(ctx, st, config.Config{}, dyn, mail.Console{Log: log}, master, auditor)
	if err != nil {
		t.Fatalf("NewSettingsHandler: %v", err)
	}

	// The JSON row no longer carries the plaintext password (but keeps the rest).
	stored, err := st.GetInstanceSetting(ctx, store.InstanceSettingSMTP)
	if err != nil {
		t.Fatal(err)
	}
	var got config.SMTP
	if err := json.Unmarshal([]byte(stored), &got); err != nil {
		t.Fatal(err)
	}
	if got.Password != "" {
		t.Fatalf("plaintext password still in settings JSON: %q", got.Password)
	}
	if got.Host != "smtp.example.com" {
		t.Fatalf("host lost in migration: %q", got.Host)
	}

	// The encrypted secret now exists and decrypts to the original password.
	enc, err := st.GetInstanceSecret(ctx, store.InstanceSecretSMTPPassword)
	if err != nil {
		t.Fatalf("encrypted secret not stored: %v", err)
	}
	plain, err := master.Decrypt(enc)
	if err != nil {
		t.Fatal(err)
	}
	if string(plain) != "s3cret" {
		t.Fatalf("decrypted secret = %q, want s3cret", plain)
	}

	// The handler still resolves the complete effective config.
	eff, source, err := h.effectiveSMTP(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if eff.Password != "s3cret" {
		t.Fatalf("effective password = %q, want s3cret", eff.Password)
	}
	if source != adminv1.SmtpSource_SMTP_SOURCE_DATABASE {
		t.Fatalf("smtp source = %v, want database", source)
	}
}
