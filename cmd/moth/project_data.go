package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
)

// exportDoc is the users-JSON document `moth project export` writes and
// `moth project import` reads. It carries each account's (possibly foreign)
// password hash so a migration off — or between — moth instances keeps users
// signed in without a password reset.
type exportDoc struct {
	Project    string       `json:"project"`
	ExportedAt time.Time    `json:"exported_at"`
	Users      []exportUser `json:"users"`
}

type exportUser struct {
	Email         string `json:"email"`
	DisplayName   string `json:"display_name,omitempty"`
	AvatarURL     string `json:"avatar_url,omitempty"`
	EmailVerified bool   `json:"email_verified"`
	Disabled      bool   `json:"disabled"`
	// CustomClaims is the JSON object embedded in the JWT `claims` claim.
	CustomClaims string `json:"custom_claims,omitempty"`
	// PasswordHash is the encoded credential; empty for social-only accounts.
	PasswordHash string `json:"password_hash,omitempty"`
	// PasswordAlgorithm is the scheme that produced PasswordHash: "argon2id"
	// for a native moth hash, or the foreign algorithm ("bcrypt", "scrypt",
	// "argon2", "pbkdf2") a migration import declared. A foreign hash is
	// verified with its original algorithm on the user's first sign-in, then
	// transparently rehashed to argon2id.
	PasswordAlgorithm string           `json:"password_algorithm,omitempty"`
	Identities        []exportIdentity `json:"identities,omitempty"`
	CreatedAt         time.Time        `json:"created_at"`
	LastLoginAt       *time.Time       `json:"last_login_at,omitempty"`
}

type exportIdentity struct {
	Provider        string `json:"provider"`
	ProviderSubject string `json:"provider_subject,omitempty"`
	Email           string `json:"email,omitempty"`
}

func newProjectExportCmd(opts *clientOpts) *cobra.Command {
	var out string
	cmd := &cobra.Command{
		Use:   "export <slug|id>",
		Short: "Export a project's users as JSON (import them with 'moth project import')",
		Long: `Export writes the project's user accounts — email, display name,
verification/disabled state, custom claims, provider identities and the
encoded password hash — as one JSON document, the input of
'moth project import'.

Password hashes travel with the users (a native argon2id hash, tagged
"argon2id"), so migrating between moth instances keeps everyone signed in
without a reset. Social identities also re-link automatically on the
user's next social sign-in. Project configuration is a separate concern:
see 'moth project dump'.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := opts.dial()
			if err != nil {
				return err
			}
			p, err := resolveProject(cmd.Context(), client, args[0])
			if err != nil {
				return err
			}
			resp, err := client.Projects.ExportProject(cmd.Context(),
				connect.NewRequest(&adminv1.ExportProjectRequest{ProjectId: p.Id}))
			if err != nil {
				return err
			}
			doc := exportDoc{Project: p.Slug, ExportedAt: time.Now().UTC(), Users: []exportUser{}}
			for _, u := range resp.Msg.Users {
				eu := exportUser{
					Email:             u.Email,
					DisplayName:       u.DisplayName,
					AvatarURL:         u.AvatarUrl,
					EmailVerified:     u.EmailVerified,
					Disabled:          u.Disabled,
					CustomClaims:      u.CustomClaims,
					PasswordHash:      u.PasswordHash,
					PasswordAlgorithm: u.PasswordAlgorithm,
				}
				if u.CreateTime != nil {
					eu.CreatedAt = u.CreateTime.AsTime()
				}
				if u.LastLoginTime != nil {
					t := u.LastLoginTime.AsTime()
					eu.LastLoginAt = &t
				}
				for _, id := range u.Identities {
					eu.Identities = append(eu.Identities, exportIdentity{
						Provider:        id.Provider,
						ProviderSubject: id.ProviderSubject,
						Email:           id.Email,
					})
				}
				doc.Users = append(doc.Users, eu)
			}
			data, err := jsonMarshalIndent(doc)
			if err != nil {
				return err
			}
			data = append(data, '\n')
			if out == "" || out == "-" {
				_, err = cmd.OutOrStdout().Write(data)
				return err
			}
			if err := os.WriteFile(out, data, 0o600); err != nil {
				return err
			}
			fmt.Printf("exported %d users of %s to %s\n", len(doc.Users), p.Slug, out)
			return nil
		},
	}
	cmd.Flags().StringVarP(&out, "output", "o", "", "output file (default: stdout)")
	return cmd
}

// importResult is the --json output of `moth project import`.
type importResult struct {
	Project string `json:"project"`
	Created int    `json:"created"`
	Skipped int    `json:"skipped"`
}

func newProjectImportCmd(opts *clientOpts) *cobra.Command {
	var file string
	var yes bool
	cmd := &cobra.Command{
		Use:   "import <slug|id> -f <export.json>",
		Short: "Import users from a 'moth project export' document (idempotent)",
		Long: `Import creates the document's users in the target project, restoring
display name, avatar, email verification, disabled state, custom claims,
provider identities and the encoded password hash. A user whose email
already exists in the project is skipped, so re-running an import is safe.

Foreign password hashes (bcrypt, scrypt, argon2, pbkdf2 — tagged per user
in the document's password_algorithm field) are accepted: each is
verified with its original algorithm on the user's first sign-in and then
transparently rehashed to argon2id, so teams can migrate from another
auth system without forcing a password reset.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := os.ReadFile(file)
			if err != nil {
				return err
			}
			var doc exportDoc
			if err := json.Unmarshal(data, &doc); err != nil {
				return fmt.Errorf("parse %s: %w", file, err)
			}
			client, err := opts.dial()
			if err != nil {
				return err
			}
			p, err := resolveProject(cmd.Context(), client, args[0])
			if err != nil {
				return err
			}
			action := fmt.Sprintf("import %d users into %q", len(doc.Users), p.Slug)
			if err := confirm(cmd, yes, action); err != nil {
				return err
			}

			req := &adminv1.ImportProjectRequest{ProjectId: p.Id}
			for _, u := range doc.Users {
				iu := &adminv1.ImportedUser{
					Email:             u.Email,
					EmailVerified:     u.EmailVerified,
					DisplayName:       u.DisplayName,
					AvatarUrl:         u.AvatarURL,
					CustomClaims:      u.CustomClaims,
					PasswordHash:      u.PasswordHash,
					PasswordAlgorithm: u.PasswordAlgorithm,
					Disabled:          u.Disabled,
				}
				for _, id := range u.Identities {
					iu.Identities = append(iu.Identities, &adminv1.ExportedIdentity{
						Provider:        id.Provider,
						ProviderSubject: id.ProviderSubject,
						Email:           id.Email,
					})
				}
				req.Users = append(req.Users, iu)
			}
			resp, err := client.Projects.ImportProject(cmd.Context(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			result := importResult{
				Project: p.Slug,
				Created: int(resp.Msg.ImportedCount),
				Skipped: int(resp.Msg.SkippedCount),
			}
			if opts.json {
				return printJSONValue(cmd, result)
			}
			fmt.Printf("project %s: %d users created, %d skipped (already exist)\n",
				p.Slug, result.Created, result.Skipped)
			return nil
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", "", "export JSON file (required)")
	cmd.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt")
	_ = cmd.MarkFlagRequired("file") // flag is registered just above
	return cmd
}
