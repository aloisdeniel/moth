// Package setup implements the provider-console orchestration behind
// `moth setup google`, `moth setup apple` and `moth doctor`: guided or
// partially automated configuration of the Google/Apple sign-in consoles
// for one moth project, always followed by verification. The commands are
// idempotent — they diff the current console/moth state against the
// desired state and only change what is needed.
//
// # Capability spike (milestone 08)
//
// Which steps automate cleanly and which stay guided, as established for
// this implementation:
//
// Google:
//   - Creating OAuth 2.0 client IDs (web/iOS/Android) has NO public API.
//     Neither the Cloud Console credentials API surface nor the IAM
//     "OAuth clients" API (workforce identity only) covers the standard
//     consent-screen clients moth needs; Firebase auto-provisions clients
//     but only inside Firebase-enrolled projects. → GUIDED: the command
//     prints the exact console URL and what to enter, then validates the
//     pasted client ID's shape before accepting it.
//   - `gcloud` (when installed and authenticated) verifies that the GCP
//     project exists before sending the user to the console. → automated,
//     best effort; without gcloud the flow is purely guided.
//   - Android signing-report fingerprints compute locally with `keytool`
//     when a keystore is at hand. → automated.
//   - Verification is unauthenticated: Google's authorization endpoint
//     distinguishes an unknown client ("invalid_client" / "The OAuth
//     client was not found") from a valid client with an unregistered
//     redirect URI ("redirect_uri_mismatch"), so client IDs — and, for
//     the web client, the registered redirect URI — are checkable without
//     credentials. → automated.
//
// Apple:
//   - Bundle IDs and their capabilities are covered by the official App
//     Store Connect API (bundleIds / bundleIdCapabilities resources,
//     JWT-authed with an ASC API key). → automated: verify/create the
//     bundle ID, enable the Sign in with Apple capability.
//   - Sign in with Apple key creation is attempted through the ASC keys
//     resource; the endpoint is not part of every documented ASC surface,
//     so a 404 from Apple degrades to the guided flow (portal URL, then
//     paste the key ID and the downloaded .p8 path). Apple serves the .p8
//     exactly once; the command uploads it into moth's encrypted provider
//     config immediately.
//   - Services IDs and their return-URL registration have NO official
//     API (fastlane's spaceship uses the unofficial portal API). →
//     GUIDED, with the exact values to paste; --unofficial-api is a
//     documented stub, deliberately not implemented.
//   - Verification: a client secret minted from the stored key is
//     dry-run against Apple's token endpoint — "invalid_grant" proves the
//     key/team/client triple is accepted, "invalid_client" proves it is
//     not. → automated whenever the key material is in hand.
//
// Every external call (Google endpoints, ASC, gcloud, keytool, prompts)
// sits behind an interface or injectable endpoint so tests run against
// doubles; nothing here talks to a real console in CI.
package setup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"connectrpc.com/connect"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/gen/moth/admin/v1/adminv1connect"
	"github.com/aloisdeniel/moth/internal/store"
)

// Status is the outcome of one verification check.
type Status string

// Check outcomes, ordered from best to worst.
const (
	StatusPass Status = "PASS"
	StatusSkip Status = "SKIP"
	StatusWarn Status = "WARN"
	StatusFail Status = "FAIL"
)

// Check is one line of the final checklist.
type Check struct {
	Name   string `json:"name"`
	Status Status `json:"status"`
	// Detail says what was observed.
	Detail string `json:"detail,omitempty"`
	// Remediation says what to do about a WARN/FAIL.
	Remediation string `json:"remediation,omitempty"`
}

// Report is the checklist a setup command or doctor run produces.
type Report struct {
	Checks []Check `json:"checks"`
}

func (r *Report) add(c Check) { r.Checks = append(r.Checks, c) }

// Pass records a successful check.
func (r *Report) Pass(name, detail string) {
	r.add(Check{Name: name, Status: StatusPass, Detail: detail})
}

// Skip records a check that did not apply.
func (r *Report) Skip(name, detail string) {
	r.add(Check{Name: name, Status: StatusSkip, Detail: detail})
}

// Warn records a non-fatal problem.
func (r *Report) Warn(name, detail, remediation string) {
	r.add(Check{Name: name, Status: StatusWarn, Detail: detail, Remediation: remediation})
}

// Fail records a fatal problem.
func (r *Report) Fail(name, detail, remediation string) {
	r.add(Check{Name: name, Status: StatusFail, Detail: detail, Remediation: remediation})
}

// Status is the overall outcome: FAIL if any check failed, else WARN if
// any warned, else PASS.
func (r *Report) Status() Status {
	overall := StatusPass
	for _, c := range r.Checks {
		switch c.Status {
		case StatusFail:
			return StatusFail
		case StatusWarn:
			overall = StatusWarn
		case StatusPass, StatusSkip:
		}
	}
	return overall
}

// Failed reports whether any check failed.
func (r *Report) Failed() bool { return r.Status() == StatusFail }

// ANSI escape sequences for the checklist; emitted only when color is on.
const (
	ansiReset  = "\x1b[0m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiRed    = "\x1b[31m"
	ansiDim    = "\x1b[2m"
)

func (s Status) glyph() (mark, color string) {
	switch s {
	case StatusPass:
		return "✓", ansiGreen
	case StatusWarn:
		return "!", ansiYellow
	case StatusFail:
		return "✗", ansiRed
	default:
		return "-", ansiDim
	}
}

// Print renders the checklist, colored when color is true.
func (r *Report) Print(w io.Writer, color bool) {
	for _, c := range r.Checks {
		mark, tint := c.Status.glyph()
		if color {
			_, _ = fmt.Fprintf(w, "  %s%s %-4s%s %s", tint, mark, c.Status, ansiReset, c.Name)
		} else {
			_, _ = fmt.Fprintf(w, "  %s %-4s %s", mark, c.Status, c.Name)
		}
		if c.Detail != "" {
			_, _ = fmt.Fprintf(w, " — %s", c.Detail)
		}
		_, _ = fmt.Fprintln(w)
		if c.Remediation != "" {
			_, _ = fmt.Fprintf(w, "           ↳ %s\n", c.Remediation)
		}
	}
	mark, tint := r.Status().glyph()
	if color {
		_, _ = fmt.Fprintf(w, "\n%s%s %s%s\n", tint, mark, r.Status(), ansiReset)
	} else {
		_, _ = fmt.Fprintf(w, "\n%s %s\n", mark, r.Status())
	}
}

// JSON renders the checklist for --json consumers.
func (r *Report) JSON() ([]byte, error) {
	return json.MarshalIndent(struct {
		Status Status  `json:"status"`
		Checks []Check `json:"checks"`
	}{Status: r.Status(), Checks: r.Checks}, "", "  ")
}

// findProjectBySlug resolves a project by its slug. The admin API is keyed
// on IDs; the CLI is keyed on slugs, so it lists and matches. The
// not-found error carries connect.CodeNotFound (wrapping store.ErrNotFound
// for the doctor's errors.Is check) so the process exits with the
// advertised not-found code.
func findProjectBySlug(ctx context.Context, pc adminv1connect.ProjectServiceClient, slug string) (*adminv1.Project, error) {
	resp, err := pc.ListProjects(ctx, connect.NewRequest(&adminv1.ListProjectsRequest{}))
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	for _, p := range resp.Msg.Projects {
		if p.Slug == slug {
			return p, nil
		}
	}
	return nil, connect.NewError(connect.CodeNotFound,
		fmt.Errorf("project %q: %w", slug, store.ErrNotFound))
}
