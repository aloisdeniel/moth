package server

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"connectrpc.com/connect"

	authrpc "github.com/aloisdeniel/moth/internal/server/rpc/auth"
	"github.com/aloisdeniel/moth/internal/store"
)

// Web-redirect OAuth fallback. OAuth consent screens are browser round
// trips, so these two legs are plain HTTP; everything stateful lives in
// authrpc (OAuthStart/OAuthCallback) next to the native flow it shares its
// identity resolution with.

// handleOAuthStart begins the flow:
// GET /oauth/{provider}/start?project={slug}&redirect={registered-scheme URI}.
func (s *Server) handleOAuthStart(w http.ResponseWriter, r *http.Request) {
	provider := r.PathValue("provider")
	project, err := s.store.GetProjectBySlug(r.Context(), r.URL.Query().Get("project"))
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	consentURL, err := s.auth.OAuthStart(r.Context(), project, provider,
		r.URL.Query().Get("redirect"))
	if err != nil {
		s.oauthErrorPage(w, r, project, err)
		return
	}
	http.Redirect(w, r, consentURL, http.StatusFound)
}

// handleOAuthCallback finishes the flow: GET with query parameters for
// Google, POST form_post for Apple. The project is recovered from the
// state's slug prefix (the full state value is still what was stored
// hashed, so a tampered prefix cannot survive the claim).
func (s *Server) handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	provider := r.PathValue("provider")
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	state := r.FormValue("state")
	slug, _, _ := strings.Cut(state, ".")
	project, err := s.store.GetProjectBySlug(r.Context(), slug)
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	if msg := r.FormValue("error"); msg != "" || r.FormValue("code") == "" {
		s.renderPageStatus(w, r, http.StatusBadRequest, project, pageData{
			Title: "Sign-in not completed",
			Error: "The provider did not complete the sign-in. Return to the app and try again.",
		})
		return
	}

	code, redirectURI, err := s.auth.OAuthCallback(r.Context(), project, provider,
		state, r.FormValue("code"), r.FormValue("user"))
	if err != nil {
		s.oauthErrorPage(w, r, project, err)
		return
	}
	if redirectURI != "" {
		http.Redirect(w, r, appendCodeParam(redirectURI, code), http.StatusFound)
		return
	}
	// No registered scheme: hosted success page (manual testing and
	// non-mobile clients); the code is shown so it can be exchanged by
	// hand.
	s.renderPage(w, r, project, pageData{
		Title:   "Signed in",
		Message: "Sign-in complete. Return to the app and exchange this one-time code: " + code,
	})
}

// oauthErrorPage maps an OAuthStart/OAuthCallback error to a friendly 4xx
// page.
func (s *Server) oauthErrorPage(w http.ResponseWriter, r *http.Request, project store.Project, err error) {
	data := pageData{Title: "Sign-in failed"}
	status := http.StatusBadRequest
	switch authrpc.ErrorReason(err) {
	case authrpc.ReasonProviderDisabled:
		data.Error = "This sign-in method is not available for this app."
	case authrpc.ReasonInvalidRedirect:
		data.Error = "The requested redirect is not registered for this app."
	case authrpc.ReasonInvalidToken:
		data.Error = "This sign-in link is invalid, expired or was already used. Return to the app and try again."
	case authrpc.ReasonInvalidProviderToken:
		status = http.StatusUnauthorized
		data.Error = "The provider sign-in could not be verified. Return to the app and try again."
	case authrpc.ReasonEmailAlreadyExists:
		data.Error = "An account with this email already exists. Sign in with it to link this provider."
	case authrpc.ReasonUserDisabled:
		status = http.StatusForbidden
		data.Error = "This account is disabled."
	case authrpc.ReasonSignupClosed:
		status = http.StatusForbidden
		data.Error = "Signup is closed for this app."
	default:
		if connect.CodeOf(err) == connect.CodeInvalidArgument {
			data.Error = "Invalid sign-in request."
		} else {
			s.internalError(w, r, err)
			return
		}
	}
	s.renderPageStatus(w, r, status, project, data)
}

// appendCodeParam adds the one-time code query parameter to the app's
// redirect URI.
func appendCodeParam(redirectURI, code string) string {
	sep := "?"
	if strings.Contains(redirectURI, "?") {
		sep = "&"
	}
	return redirectURI + sep + "code=" + url.QueryEscape(code)
}
