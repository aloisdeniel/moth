package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/internal/token"
)

// exportDoc is the users-JSON document `moth project export` writes and
// `moth project import` reads. Milestone 10's migration format extends it
// (foreign password hashes); until then credentials never round-trip.
type exportDoc struct {
	Project    string       `json:"project"`
	ExportedAt time.Time    `json:"exported_at"`
	Users      []exportUser `json:"users"`
}

type exportUser struct {
	Email         string `json:"email"`
	DisplayName   string `json:"display_name,omitempty"`
	EmailVerified bool   `json:"email_verified"`
	Disabled      bool   `json:"disabled"`
	// CustomClaims is the JSON object embedded in the JWT `claims` claim.
	CustomClaims string `json:"custom_claims,omitempty"`
	// Providers is informational ("password", "google", "apple"): social
	// identities re-link on the user's next social sign-in and password
	// credentials cannot be exported.
	Providers []string  `json:"providers,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func newProjectExportCmd(opts *clientOpts) *cobra.Command {
	var out string
	cmd := &cobra.Command{
		Use:   "export <slug|id>",
		Short: "Export a project's users as JSON (import them with 'moth project import')",
		Long: `Export writes the project's user accounts — email, display name,
verification/disabled state, custom claims — as one JSON document, the
input of 'moth project import'.

Credentials never leave the server: password hashes are not exported (the
milestone-10 migration format will carry foreign hashes), and social
identities re-link automatically on the user's next social sign-in.
Project configuration is a separate concern: see 'moth project dump'.`,
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
			doc := exportDoc{Project: p.Slug, ExportedAt: time.Now().UTC(), Users: []exportUser{}}
			pageToken := ""
			for {
				resp, err := client.Users.ListUsers(cmd.Context(), connect.NewRequest(&adminv1.ListUsersRequest{
					ProjectId: p.Id, PageSize: 200, PageToken: pageToken,
				}))
				if err != nil {
					return err
				}
				for _, u := range resp.Msg.Users {
					doc.Users = append(doc.Users, exportUser{
						Email:         u.Email,
						DisplayName:   u.DisplayName,
						EmailVerified: u.EmailVerified,
						Disabled:      u.Disabled,
						CustomClaims:  u.CustomClaims,
						Providers:     u.Providers,
						CreatedAt:     u.CreateTime.AsTime(),
					})
				}
				if pageToken = resp.Msg.NextPageToken; pageToken == "" {
					break
				}
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
	var invite, yes bool
	cmd := &cobra.Command{
		Use:   "import <slug|id> -f <export.json>",
		Short: "Import users from a 'moth project export' document (idempotent)",
		Long: `Import creates the document's users in the target project, restoring
display name, email verification, disabled state and custom claims. A
user whose email already exists in the project is skipped, so re-running
an import is safe.

Passwords do not round-trip (moth never exports hashes): pass --invite to
send each newly created user a set-password email; without it each user
gets an unusable random password and recovers through "forgot password"
or a social sign-in, which re-links automatically.`,
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

			result := importResult{Project: p.Slug}
			for _, u := range doc.Users {
				req := &adminv1.CreateUserRequest{
					ProjectId: p.Id, Email: u.Email, DisplayName: u.DisplayName,
					EmailVerified: u.EmailVerified, SendInvite: invite,
				}
				if !invite {
					// Hashes never round-trip; an unusable random password
					// creates the account, recovered via reset or social.
					req.Password = token.Random(32)
				}
				created, err := client.Users.CreateUser(cmd.Context(), connect.NewRequest(req))
				if connect.CodeOf(err) == connect.CodeAlreadyExists {
					result.Skipped++
					continue
				}
				if err != nil {
					return fmt.Errorf("create %s: %w", u.Email, err)
				}
				result.Created++
				id := created.Msg.User.Id
				if u.CustomClaims != "" && u.CustomClaims != "{}" {
					if _, err := client.Users.UpdateUser(cmd.Context(), connect.NewRequest(&adminv1.UpdateUserRequest{
						ProjectId: p.Id, UserId: id,
						User:       &adminv1.User{CustomClaims: u.CustomClaims},
						UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"custom_claims"}},
					})); err != nil {
						return fmt.Errorf("restore claims of %s: %w", u.Email, err)
					}
				}
				if u.Disabled {
					if _, err := client.Users.DisableUser(cmd.Context(), connect.NewRequest(&adminv1.DisableUserRequest{
						ProjectId: p.Id, UserId: id,
					})); err != nil {
						return fmt.Errorf("disable %s: %w", u.Email, err)
					}
				}
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
	cmd.Flags().BoolVar(&invite, "invite", false, "send each created user a set-password invite email")
	cmd.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt")
	_ = cmd.MarkFlagRequired("file") // flag is registered just above
	return cmd
}
