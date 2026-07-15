package mail

import (
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestEmailsRender(t *testing.T) {
	msgs := map[string]Message{
		"verification":  Verification("My App", "u@example.com", "https://moth/p/app/verify?token=t"),
		"reset":         PasswordReset("My App", "u@example.com", "https://moth/p/app/reset?token=t"),
		"change":        EmailChangeConfirm("My App", "new@example.com", "https://moth/p/app/confirm-email?token=t"),
		"changedNotice": EmailChangedNotice("My App", "old@example.com", "new@example.com", "https://moth/p/app/confirm-email?token=r"),
		"accountExists": AccountExists("My App", "u@example.com"),
	}
	for name, m := range msgs {
		if m.To == "" || m.Subject == "" || m.Text == "" || m.HTML == "" {
			t.Errorf("%s: incomplete message: %+v", name, m)
		}
		if !strings.Contains(m.Text, "My App") {
			t.Errorf("%s: text body does not mention the project", name)
		}
	}
	if !strings.Contains(msgs["changedNotice"].Text, "72 hours") {
		t.Error("changed notice must mention the revert window")
	}
	if !strings.Contains(msgs["verification"].HTML, `href="https://moth/p/app/verify?token=t"`) {
		t.Error("verification HTML must link the verify URL")
	}
}

func TestBuildMIME(t *testing.T) {
	raw, err := buildMIME("moth@example.com", Message{
		To: "u@example.com", Subject: "Hi\r\nInjected: x", Text: "hello", HTML: "<b>hello</b>",
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if strings.Contains(s, "Injected: x\r\n") && strings.Contains(s, "Subject: Hi\r\nInjected") {
		t.Error("subject header injection was not neutralized")
	}
	for _, want := range []string{"From: moth@example.com", "To: u@example.com",
		"multipart/alternative", "text/plain", "text/html"} {
		if !strings.Contains(s, want) {
			t.Errorf("MIME message missing %q", want)
		}
	}
}

func TestConsoleTransport(t *testing.T) {
	var sb strings.Builder
	c := Console{Log: slog.New(slog.NewTextHandler(&sb, nil))}
	if err := c.Send(context.Background(), Verification("My App", "u@example.com", "https://link")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sb.String(), "https://link") {
		t.Error("console transport must log the full email including links")
	}
}
