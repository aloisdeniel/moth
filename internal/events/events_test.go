package events

import (
	"context"
	"testing"
)

func TestConstructorsPickUpClientInfo(t *testing.T) {
	ctx := WithClientInfo(context.Background(),
		ClientInfo{Platform: PlatformIOS, SDKVersion: "1.2.3"})

	e := Login(ctx, "p1", "u1", "google")
	if e.Type != TypeUserLogin || e.ProjectID != "p1" || e.UserID != "u1" || e.Provider != "google" {
		t.Fatalf("Login = %+v", e)
	}
	if e.Platform != PlatformIOS || e.SDKVersion != "1.2.3" {
		t.Fatalf("client info not stamped: %+v", e)
	}
	if e.CreatedAt.IsZero() {
		t.Fatal("CreatedAt not stamped")
	}

	// Without client info in ctx the fields stay empty.
	e = Signup(context.Background(), "p1", "u1", "")
	if e.Platform != "" || e.SDKVersion != "" {
		t.Fatalf("unexpected client info: %+v", e)
	}
}

func TestLoginFailedCarriesNoUserID(t *testing.T) {
	ctx := context.Background()

	e := LoginFailed(ctx, "p1", "apple", ReasonInvalidCredentials)
	if e.UserID != "" {
		t.Fatalf("login_failed carries user id %q", e.UserID)
	}
	if e.Type != TypeUserLoginFailed || e.Provider != "apple" {
		t.Fatalf("LoginFailed = %+v", e)
	}
	if got := e.Metadata["reason"]; got != string(ReasonInvalidCredentials) {
		t.Fatalf("reason = %q", got)
	}

	// Unknown reasons are bucketed, never stored verbatim.
	e = LoginFailed(ctx, "p1", "", FailureReason("password was hunter2"))
	if got := e.Metadata["reason"]; got != string(ReasonOther) {
		t.Fatalf("unknown reason bucketed to %q, want %q", got, ReasonOther)
	}
}

func TestMetadataJSON(t *testing.T) {
	if got := (Event{}).MetadataJSON(); got != "" {
		t.Fatalf("empty metadata encoded to %q, want \"\"", got)
	}

	e := Event{Metadata: map[string]string{"reason": "disabled"}}
	if got := e.MetadataJSON(); got != `{"reason":"disabled"}` {
		t.Fatalf("MetadataJSON = %s", got)
	}
}

func TestAllConstructorsSetTheirType(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		got  Event
		want string
	}{
		{Signup(ctx, "p", "u", ""), TypeUserSignup},
		{Login(ctx, "p", "u", ""), TypeUserLogin},
		{TokenRefresh(ctx, "p", "u"), TypeTokenRefresh},
		{LoginFailed(ctx, "p", "", ReasonOther), TypeUserLoginFailed},
		{PasswordResetCompleted(ctx, "p", "u"), TypePasswordResetCompleted},
		{EmailVerified(ctx, "p", "u"), TypeEmailVerified},
		{UserDeleted(ctx, "p", "u"), TypeUserDeleted},
		{IdentityLinked(ctx, "p", "u", "google"), TypeIdentityLinked},
	}
	for _, c := range cases {
		if c.got.Type != c.want {
			t.Errorf("constructor produced type %q, want %q", c.got.Type, c.want)
		}
		if c.got.ProjectID != "p" {
			t.Errorf("%s: project id %q", c.want, c.got.ProjectID)
		}
	}
}
