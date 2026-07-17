package server

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"connectrpc.com/connect"

	"github.com/aloisdeniel/moth/internal/i18n"
	authrpc "github.com/aloisdeniel/moth/internal/server/rpc/auth"
	"github.com/aloisdeniel/moth/internal/store"
)

// Web-redirect OAuth fallback. OAuth consent screens are browser round
// trips, so these two legs are plain HTTP; everything stateful lives in
// authrpc (OAuthStart/OAuthCallback) next to the native flow it shares its
// identity resolution with.

// handleOAuthStart begins the flow:
// GET /oauth/{provider}/start?project={slug}&redirect={URI}, where the
// redirect URI targets a registered custom scheme (mobile deep link) or a
// registered web origin (browser SPA).
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
	loc := s.localize(r, project)
	if msg := r.FormValue("error"); msg != "" || r.FormValue("code") == "" {
		s.renderPageStatus(w, r, http.StatusBadRequest, project, pageData{
			Title:   loc.Value(i18n.HostedOAuthIncompleteTitle, nil),
			Error:   loc.Value(i18n.HostedOAuthIncomplete, nil),
			IsError: true,
		}, loc)
		return
	}

	code, redirectURI, err := s.auth.OAuthCallback(r.Context(), project, provider,
		state, r.FormValue("code"), r.FormValue("user"))
	if err != nil {
		s.oauthErrorPage(w, r, project, err)
		return
	}
	if redirectURI != "" {
		// A plain 302 works for both registered destinations: custom
		// schemes hand off to the mobile app, http(s) origins land the
		// browser back on the SPA's return route with ?code=….
		http.Redirect(w, r, appendCodeParam(redirectURI, code), http.StatusFound)
		return
	}
	// No redirect registered at start: hosted success page (manual testing
	// and non-mobile clients); the code is shown so it can be exchanged by
	// hand.
	s.renderPage(w, r, project, pageData{
		Title:   loc.Value(i18n.HostedOAuthSuccess, map[string]string{"app": project.Name}),
		Message: loc.Value(i18n.HostedOAuthCode, map[string]string{"code": code}),
	}, loc)
}

// oauthErrorPage maps an OAuthStart/OAuthCallback error to a friendly 4xx
// page.
func (s *Server) oauthErrorPage(w http.ResponseWriter, r *http.Request, project store.Project, err error) {
	loc := s.localize(r, project)
	data := pageData{Title: loc.Value(i18n.HostedOAuthFailedTitle, nil), IsError: true}
	status := http.StatusBadRequest
	switch authrpc.ErrorReason(err) {
	case authrpc.ReasonProviderDisabled:
		data.Error = loc.Value(i18n.HostedOAuthErrProviderDisabled, nil)
	case authrpc.ReasonInvalidRedirect:
		data.Error = loc.Value(i18n.HostedOAuthErrInvalidRedirect, nil)
	case authrpc.ReasonInvalidToken:
		data.Error = loc.Value(i18n.HostedOAuthErrInvalidToken, nil)
	case authrpc.ReasonInvalidProviderToken:
		status = http.StatusUnauthorized
		data.Error = loc.Value(i18n.HostedOAuthErrProviderToken, nil)
	case authrpc.ReasonEmailAlreadyExists:
		data.Error = loc.Value(i18n.HostedOAuthErrEmailExists, nil)
	case authrpc.ReasonUserDisabled:
		status = http.StatusForbidden
		data.Error = loc.Value(i18n.HostedOAuthErrUserDisabled, nil)
	case authrpc.ReasonSignupClosed:
		status = http.StatusForbidden
		data.Error = loc.Value(i18n.HostedOAuthErrSignupClosed, nil)
	default:
		if connect.CodeOf(err) == connect.CodeInvalidArgument {
			data.Error = loc.Value(i18n.HostedOAuthErrInvalid, nil)
		} else {
			s.internalError(w, r, err)
			return
		}
	}
	s.renderPageStatus(w, r, status, project, data, loc)
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
