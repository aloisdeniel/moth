package server

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"

	"connectrpc.com/connect"
	gojwt "github.com/golang-jwt/jwt/v5"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/gen/moth/admin/v1/adminv1connect"
	authv1 "github.com/aloisdeniel/moth/gen/moth/auth/v1"
	serverv1 "github.com/aloisdeniel/moth/gen/moth/server/v1"
)

// jwksKeyfunc verifies tokens against a JWKS snapshot by kid.
func jwksKeyfunc(keys map[string]*ecdsa.PublicKey) gojwt.Keyfunc {
	return func(tok *gojwt.Token) (any, error) {
		kid, _ := tok.Header["kid"].(string)
		if key, ok := keys[kid]; ok {
			return key, nil
		}
		return nil, gojwt.ErrTokenUnverifiable
	}
}

func (e *testEnv) adminAccounts() adminv1connect.AdminAccountServiceClient {
	return adminv1connect.NewAdminAccountServiceClient(e.client, e.url)
}

func (e *testEnv) adminSettings() adminv1connect.InstanceSettingsServiceClient {
	return adminv1connect.NewInstanceSettingsServiceClient(e.client, e.url)
}

// TestResetSigningKeyKillsEveryToken covers the milestone acceptance
// criterion: after a signing-key reset, previously issued JWTs no longer
// validate against the JWKS, IntrospectToken reports them invalid, and all
// refresh tokens are dead.
func TestResetSigningKeyKillsEveryToken(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	p, sk := e.createProject(t, "App")
	auth := e.authClient(p.PublishableKey)
	ctx := context.Background()

	su, err := auth.SignUp(ctx, connect.NewRequest(&authv1.SignUpRequest{
		Email: "u@example.com", Password: "password-1"}))
	if err != nil {
		t.Fatal(err)
	}
	access, refresh := su.Msg.Tokens.AccessToken, su.Msg.Tokens.RefreshToken

	// Sanity: the fresh token validates and introspects as active.
	oldKeys := e.jwksKeys(t, p.Slug)
	if _, err := gojwt.Parse(access, jwksKeyfunc(oldKeys),
		gojwt.WithValidMethods([]string{"ES256"})); err != nil {
		t.Fatalf("fresh token should validate: %v", err)
	}

	key, err := e.projects.GetSigningKey(ctx, connect.NewRequest(
		&adminv1.GetSigningKeyRequest{ProjectId: p.Id}))
	if err != nil {
		t.Fatal(err)
	}
	if key.Msg.Audience != p.Slug ||
		key.Msg.JwksUrl != "http://localhost:8080/p/"+p.Slug+"/.well-known/jwks.json" ||
		key.Msg.Issuer != "http://localhost:8080/p/"+p.Slug ||
		!strings.Contains(key.Msg.Key.PublicKeyPem, "BEGIN PUBLIC KEY") {
		t.Fatalf("signing key card values: %+v", key.Msg)
	}
	oldKid := key.Msg.Key.Kid

	reset, err := e.projects.ResetSigningKey(ctx, connect.NewRequest(
		&adminv1.ResetSigningKeyRequest{ProjectId: p.Id}))
	if err != nil {
		t.Fatal(err)
	}
	if reset.Msg.Key.Kid == oldKid {
		t.Fatal("reset must mint a different keypair")
	}

	// The old key is gone from the JWKS immediately.
	newKeys := e.jwksKeys(t, p.Slug)
	if _, ok := newKeys[oldKid]; ok {
		t.Fatal("old kid still served in JWKS after reset")
	}
	if _, ok := newKeys[reset.Msg.Key.Kid]; !ok {
		t.Fatal("new kid missing from JWKS")
	}
	if _, err := gojwt.Parse(access, jwksKeyfunc(newKeys),
		gojwt.WithValidMethods([]string{"ES256"})); err == nil {
		t.Fatal("old JWT still validates against the JWKS after reset")
	}

	// Introspection reports the old token invalid.
	intro, err := e.tokenClient(sk).IntrospectToken(ctx, connect.NewRequest(
		&serverv1.IntrospectTokenRequest{AccessToken: access}))
	if err != nil {
		t.Fatal(err)
	}
	if intro.Msg.Active {
		t.Fatal("old JWT introspects as active after reset")
	}

	// Every refresh token is dead: the user must sign in again.
	if _, err := auth.RefreshToken(ctx, connect.NewRequest(&authv1.RefreshTokenRequest{
		RefreshToken: refresh})); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("refresh after reset: %v, want unauthenticated", err)
	}
	si, err := auth.SignIn(ctx, connect.NewRequest(&authv1.SignInRequest{
		Email: "u@example.com", Password: "password-1"}))
	if err != nil {
		t.Fatalf("sign in after reset: %v", err)
	}
	if _, err := gojwt.Parse(si.Msg.Tokens.AccessToken, jwksKeyfunc(newKeys),
		gojwt.WithValidMethods([]string{"ES256"})); err != nil {
		t.Fatalf("new token should validate against new JWKS: %v", err)
	}
}

func TestRegenerateSecretKey(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	p, oldSK := e.createProject(t, "App")
	ctx := context.Background()

	regen, err := e.projects.RegenerateSecretKey(ctx, connect.NewRequest(
		&adminv1.RegenerateSecretKeyRequest{ProjectId: p.Id}))
	if err != nil {
		t.Fatal(err)
	}
	newSK := regen.Msg.SecretKey
	if newSK == "" || newSK == oldSK {
		t.Fatalf("regenerated key: %q", newSK)
	}

	// Old key is dead, new key works.
	_, err = e.userClient(oldSK).ListUsers(ctx, connect.NewRequest(&serverv1.ListUsersRequest{}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("old sk: %v, want unauthenticated", err)
	}
	if _, err := e.userClient(newSK).ListUsers(ctx, connect.NewRequest(&serverv1.ListUsersRequest{})); err != nil {
		t.Fatalf("new sk: %v", err)
	}
}

func TestAdminUsersPaginationAndSearch(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	p, sk := e.createProject(t, "App")
	users := e.userClient(sk)
	admins := e.adminUsers()
	ctx := context.Background()

	for i := 1; i <= 5; i++ {
		_, err := users.CreateUser(ctx, connect.NewRequest(&serverv1.CreateUserRequest{
			Email:       fmt.Sprintf("user%d@example.com", i),
			DisplayName: fmt.Sprintf("User %d", i),
			Password:    "password-1",
		}))
		if err != nil {
			t.Fatal(err)
		}
	}

	// Page through newest-first, two per page.
	var seen []string
	pageToken := ""
	for {
		page, err := admins.ListUsers(ctx, connect.NewRequest(&adminv1.ListUsersRequest{
			ProjectId: p.Id, PageSize: 2, PageToken: pageToken}))
		if err != nil {
			t.Fatal(err)
		}
		if page.Msg.TotalSize != 5 {
			t.Fatalf("total_size = %d, want 5", page.Msg.TotalSize)
		}
		for _, u := range page.Msg.Users {
			seen = append(seen, u.Email)
			if len(u.Providers) != 1 || u.Providers[0] != "password" {
				t.Fatalf("providers of %s: %v", u.Email, u.Providers)
			}
		}
		if page.Msg.NextPageToken == "" {
			break
		}
		pageToken = page.Msg.NextPageToken
	}
	want := []string{"user5@example.com", "user4@example.com", "user3@example.com",
		"user2@example.com", "user1@example.com"}
	if strings.Join(seen, ",") != strings.Join(want, ",") {
		t.Fatalf("pages walked: %v, want %v", seen, want)
	}

	// Substring search on email and display name.
	res, err := admins.ListUsers(ctx, connect.NewRequest(&adminv1.ListUsersRequest{
		ProjectId: p.Id, Query: "user3"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Msg.Users) != 1 || res.Msg.Users[0].Email != "user3@example.com" || res.Msg.TotalSize != 1 {
		t.Fatalf("search: %+v", res.Msg)
	}
	// LIKE wildcards in the query must be literal.
	res, err = admins.ListUsers(ctx, connect.NewRequest(&adminv1.ListUsersRequest{
		ProjectId: p.Id, Query: "%"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Msg.Users) != 0 {
		t.Fatalf("wildcard query must match nothing, got %d users", len(res.Msg.Users))
	}
}

func TestAdminCreateUserWithPasswordAndInvite(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	p, _ := e.createProject(t, "App")
	admins := e.adminUsers()
	auth := e.authClient(p.PublishableKey)
	ctx := context.Background()

	// Neither password nor invite → invalid.
	_, err := admins.CreateUser(ctx, connect.NewRequest(&adminv1.CreateUserRequest{
		ProjectId: p.Id, Email: "nobody@example.com"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("no password, no invite: %v", err)
	}

	// With password: the user can sign in immediately.
	created, err := admins.CreateUser(ctx, connect.NewRequest(&adminv1.CreateUserRequest{
		ProjectId: p.Id, Email: "direct@example.com", Password: "password-1",
		EmailVerified: true}))
	if err != nil {
		t.Fatal(err)
	}
	if !created.Msg.User.EmailVerified || created.Msg.User.Providers[0] != "password" {
		t.Fatalf("created user: %+v", created.Msg.User)
	}
	if _, err := auth.SignIn(ctx, connect.NewRequest(&authv1.SignInRequest{
		Email: "direct@example.com", Password: "password-1"})); err != nil {
		t.Fatalf("sign in as admin-created user: %v", err)
	}

	// With invite: a set-password email goes out; completing it via the
	// public ConfirmPasswordReset RPC establishes the password.
	if _, err := admins.CreateUser(ctx, connect.NewRequest(&adminv1.CreateUserRequest{
		ProjectId: p.Id, Email: "invited@example.com", SendInvite: true})); err != nil {
		t.Fatal(err)
	}
	link := e.lastLinkTo(t, "invited@example.com")
	u, err := url.Parse(link)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(u.Path, "/p/"+p.Slug+"/reset") {
		t.Fatalf("invite link should use the hosted reset page: %s", link)
	}
	if _, err := auth.ConfirmPasswordReset(ctx, connect.NewRequest(&authv1.ConfirmPasswordResetRequest{
		Token: u.Query().Get("token"), NewPassword: "chosen-by-user-1"})); err != nil {
		t.Fatal(err)
	}
	if _, err := auth.SignIn(ctx, connect.NewRequest(&authv1.SignInRequest{
		Email: "invited@example.com", Password: "chosen-by-user-1"})); err != nil {
		t.Fatalf("sign in after invite: %v", err)
	}
}

func TestAdminUserDetailUpdateAndSessions(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	p, _ := e.createProject(t, "App")
	admins := e.adminUsers()
	auth := e.authClient(p.PublishableKey)
	ctx := context.Background()

	su, err := auth.SignUp(ctx, connect.NewRequest(&authv1.SignUpRequest{
		Email: "u@example.com", Password: "password-1", DeviceInfo: "iPhone 17"}))
	if err != nil {
		t.Fatal(err)
	}
	userID := su.Msg.User.Id
	// A second device session.
	if _, err := auth.SignIn(ctx, connect.NewRequest(&authv1.SignInRequest{
		Email: "u@example.com", Password: "password-1", DeviceInfo: "Pixel 12"})); err != nil {
		t.Fatal(err)
	}

	detail, err := admins.GetUser(ctx, connect.NewRequest(&adminv1.GetUserRequest{
		ProjectId: p.Id, UserId: userID}))
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Msg.Sessions) != 2 {
		t.Fatalf("sessions: %+v", detail.Msg.Sessions)
	}
	if detail.Msg.User.LastLoginTime == nil {
		t.Fatal("last_login_time should be set after a sign-in")
	}

	// Update custom claims through the FieldMask; invalid JSON is refused.
	_, err = admins.UpdateUser(ctx, connect.NewRequest(&adminv1.UpdateUserRequest{
		ProjectId: p.Id, UserId: userID,
		User:       &adminv1.User{CustomClaims: "not json"},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"custom_claims"}},
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("invalid claims: %v", err)
	}
	upd, err := admins.UpdateUser(ctx, connect.NewRequest(&adminv1.UpdateUserRequest{
		ProjectId: p.Id, UserId: userID,
		User:       &adminv1.User{CustomClaims: `{"role": "admin"}`, DisplayName: "ignored"},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"custom_claims"}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if upd.Msg.User.CustomClaims != `{"role":"admin"}` {
		t.Fatalf("claims: %s", upd.Msg.User.CustomClaims)
	}
	if upd.Msg.User.DisplayName != "" {
		t.Fatal("display_name updated although not in the mask")
	}

	// Revoke all sessions: both devices die.
	rev, err := admins.RevokeUserSessions(ctx, connect.NewRequest(&adminv1.RevokeUserSessionsRequest{
		ProjectId: p.Id, UserId: userID}))
	if err != nil {
		t.Fatal(err)
	}
	if rev.Msg.RevokedCount != 2 {
		t.Fatalf("revoked_count = %d, want 2", rev.Msg.RevokedCount)
	}
	if _, err := auth.RefreshToken(ctx, connect.NewRequest(&authv1.RefreshTokenRequest{
		RefreshToken: su.Msg.Tokens.RefreshToken})); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("refresh after revoke: %v", err)
	}

	// Forced password reset email.
	if _, err := admins.SendPasswordReset(ctx, connect.NewRequest(&adminv1.SendPasswordResetRequest{
		ProjectId: p.Id, UserId: userID})); err != nil {
		t.Fatal(err)
	}
	if link := e.lastLinkTo(t, "u@example.com"); !strings.Contains(link, "/reset?token=") {
		t.Fatalf("reset link: %s", link)
	}

	// Delete removes the account entirely.
	if _, err := admins.DeleteUser(ctx, connect.NewRequest(&adminv1.DeleteUserRequest{
		ProjectId: p.Id, UserId: userID})); err != nil {
		t.Fatal(err)
	}
	_, err = admins.GetUser(ctx, connect.NewRequest(&adminv1.GetUserRequest{
		ProjectId: p.Id, UserId: userID}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("get after delete: %v", err)
	}
}

func TestAdminInviteFlow(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	accounts := e.adminAccounts()
	ctx := context.Background()

	// Inviting an existing admin is refused.
	_, err := accounts.InviteAdmin(ctx, connect.NewRequest(&adminv1.InviteAdminRequest{
		Email: "ops@example.com"}))
	if connect.CodeOf(err) != connect.CodeAlreadyExists {
		t.Fatalf("invite existing admin: %v", err)
	}

	inv, err := accounts.InviteAdmin(ctx, connect.NewRequest(&adminv1.InviteAdminRequest{
		Email: "second@example.com"}))
	if err != nil {
		t.Fatal(err)
	}
	if inv.Msg.Emailed {
		t.Fatal("no SMTP configured: invite must not claim it was emailed")
	}
	inviteURL, err := url.Parse(inv.Msg.InviteUrl)
	if err != nil || inviteURL.Query().Get("token") == "" {
		t.Fatalf("invite URL: %q (%v)", inv.Msg.InviteUrl, err)
	}
	tok := inviteURL.Query().Get("token")

	list, err := accounts.ListAdminInvites(ctx, connect.NewRequest(&adminv1.ListAdminInvitesRequest{}))
	if err != nil || len(list.Msg.Invites) != 1 {
		t.Fatalf("list invites: %v %+v", err, list)
	}

	// A fresh browser accepts the invite: weak password refused, then the
	// account is created and signed in via cookie.
	jar, _ := cookiejar.New(nil)
	fresh := &http.Client{Jar: jar}
	freshAccounts := adminv1connect.NewAdminAccountServiceClient(fresh, e.url)
	_, err = freshAccounts.AcceptAdminInvite(ctx, connect.NewRequest(&adminv1.AcceptAdminInviteRequest{
		Token: tok, Password: "short"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("weak invite password: %v", err)
	}
	acc, err := freshAccounts.AcceptAdminInvite(ctx, connect.NewRequest(&adminv1.AcceptAdminInviteRequest{
		Token: tok, Password: "second-password"}))
	if err != nil {
		t.Fatal(err)
	}
	if acc.Msg.Admin.Email != "second@example.com" {
		t.Fatalf("accepted admin: %+v", acc.Msg.Admin)
	}
	freshProjects := adminv1connect.NewProjectServiceClient(fresh, e.url)
	if _, err := freshProjects.ListProjects(ctx, connect.NewRequest(&adminv1.ListProjectsRequest{})); err != nil {
		t.Fatalf("invited admin should be signed in: %v", err)
	}

	// The invite is single-use.
	_, err = freshAccounts.AcceptAdminInvite(ctx, connect.NewRequest(&adminv1.AcceptAdminInviteRequest{
		Token: tok, Password: "another-password"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("second accept: %v", err)
	}

	admins, err := accounts.ListAdmins(ctx, connect.NewRequest(&adminv1.ListAdminsRequest{}))
	if err != nil || len(admins.Msg.Admins) != 2 {
		t.Fatalf("admins after invite: %v %+v", err, admins)
	}
}

func TestAdminChangePassword(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	accounts := e.adminAccounts()
	ctx := context.Background()

	_, err := accounts.ChangePassword(ctx, connect.NewRequest(&adminv1.ChangePasswordRequest{
		CurrentPassword: "wrong", NewPassword: "new-password-1"}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("wrong current password: %v", err)
	}
	if _, err := accounts.ChangePassword(ctx, connect.NewRequest(&adminv1.ChangePasswordRequest{
		CurrentPassword: "hunter22", NewPassword: "new-password-1"})); err != nil {
		t.Fatal(err)
	}

	// The current session survives; a new login needs the new password.
	if _, err := e.projects.ListProjects(ctx, connect.NewRequest(&adminv1.ListProjectsRequest{})); err != nil {
		t.Fatalf("current session after password change: %v", err)
	}
	_, err = e.sessions.Login(ctx, connect.NewRequest(&adminv1.LoginRequest{
		Email: "ops@example.com", Password: "hunter22"}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("old password: %v", err)
	}
	if _, err := e.sessions.Login(ctx, connect.NewRequest(&adminv1.LoginRequest{
		Email: "ops@example.com", Password: "new-password-1"})); err != nil {
		t.Fatal(err)
	}
}

func TestInstanceSettingsAndTestEmail(t *testing.T) {
	e := newTestEnv(t, "tok")
	e.setup(t, "tok")
	settings := e.adminSettings()
	ctx := context.Background()

	got, err := settings.GetInstanceSettings(ctx, connect.NewRequest(&adminv1.GetInstanceSettingsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if got.Msg.BaseUrl != "http://localhost:8080" ||
		got.Msg.SmtpSource != adminv1.SmtpSource_SMTP_SOURCE_NONE {
		t.Fatalf("instance settings: %+v", got.Msg)
	}

	// With no SMTP anywhere, a test email goes through the fallback
	// transport (the capture mailer in tests, the console in production).
	if _, err := settings.SendTestEmail(ctx, connect.NewRequest(&adminv1.SendTestEmailRequest{
		To: "ops@example.com"})); err != nil {
		t.Fatal(err)
	}
	if msg := e.mails.lastTo(t, "ops@example.com"); !strings.Contains(msg.Subject, "test email") {
		t.Fatalf("test email subject: %q", msg.Subject)
	}

	// Storing SMTP settings flips the source to database; the password is
	// write-only.
	upd, err := settings.UpdateSmtpSettings(ctx, connect.NewRequest(&adminv1.UpdateSmtpSettingsRequest{
		Smtp: &adminv1.SmtpSettings{
			Host: "mail.example.com", Username: "mailer",
			Password: "secret", From: "noreply@example.com"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if upd.Msg.SmtpSource != adminv1.SmtpSource_SMTP_SOURCE_DATABASE ||
		upd.Msg.Smtp.Port != 587 || upd.Msg.Smtp.Password != "" || !upd.Msg.SmtpHasPassword {
		t.Fatalf("updated smtp: %+v", upd.Msg)
	}

	// An invalid sender is refused.
	_, err = settings.UpdateSmtpSettings(ctx, connect.NewRequest(&adminv1.UpdateSmtpSettingsRequest{
		Smtp: &adminv1.SmtpSettings{Host: "mail.example.com", From: "not-an-email"},
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("invalid from: %v", err)
	}

	// Clearing the host falls back to none/console.
	cleared, err := settings.UpdateSmtpSettings(ctx, connect.NewRequest(&adminv1.UpdateSmtpSettingsRequest{
		Smtp: &adminv1.SmtpSettings{Host: ""},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if cleared.Msg.SmtpSource != adminv1.SmtpSource_SMTP_SOURCE_NONE {
		t.Fatalf("cleared smtp source: %v", cleared.Msg.SmtpSource)
	}
	if _, err := settings.SendTestEmail(ctx, connect.NewRequest(&adminv1.SendTestEmailRequest{
		To: "ops@example.com"})); err != nil {
		t.Fatalf("test email after clearing: %v", err)
	}
}

func TestProtoFilesServed(t *testing.T) {
	e := newTestEnv(t, "")
	for _, p := range []string{
		"/protos/moth/auth/v1/auth.proto",
		"/protos/moth/server/v1/token.proto",
		"/protos/moth/server/v1/user.proto",
	} {
		resp, err := e.client.Get(e.url + p)
		if err != nil {
			t.Fatal(err)
		}
		body := make([]byte, 64)
		n, _ := resp.Body.Read(body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK || !strings.Contains(string(body[:n]), "syntax = \"proto3\"") {
			t.Fatalf("GET %s: %d %q", p, resp.StatusCode, body[:n])
		}
	}
	resp, err := e.client.Get(e.url + "/protos/moth/admin/v1/../../../etc/passwd")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Fatal("path traversal must not be served")
	}
}
