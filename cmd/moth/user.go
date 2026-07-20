package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/internal/cli"
)

func newUserCmd() *cobra.Command {
	var opts clientOpts
	var project string
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage a project's end users (remote)",
	}
	addClientFlags(cmd, &opts)
	cmd.PersistentFlags().StringVar(&project, "project", "", "project slug or id (required)")
	_ = cmd.MarkPersistentFlagRequired("project") // flag is registered just above
	cmd.AddCommand(
		newUserListCmd(&opts, &project),
		newUserGetCmd(&opts, &project),
		newUserCreateCmd(&opts, &project),
		newUserInviteCmd(&opts, &project),
		newUserDisableCmd(&opts, &project),
		newUserEnableCmd(&opts, &project),
		newUserDeleteCmd(&opts, &project),
		newUserClaimsCmd(&opts, &project),
		newUserSessionsCmd(&opts, &project),
	)
	return cmd
}

// resolveUser finds a user by id or email inside a project. Emails go
// through the list query; ids straight to GetUser.
func resolveUser(ctx context.Context, client *cli.Client, projectID, ref string) (*adminv1.GetUserResponse, error) {
	if strings.Contains(ref, "@") {
		list, err := client.Users.ListUsers(ctx, connect.NewRequest(&adminv1.ListUsersRequest{
			ProjectId: projectID, Query: ref, PageSize: 200,
		}))
		if err != nil {
			return nil, err
		}
		for _, u := range list.Msg.Users {
			if strings.EqualFold(u.Email, ref) {
				resp, err := client.Users.GetUser(ctx, connect.NewRequest(&adminv1.GetUserRequest{
					ProjectId: projectID, UserId: u.Id,
				}))
				if err != nil {
					return nil, err
				}
				return resp.Msg, nil
			}
		}
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("user %q not found", ref))
	}
	resp, err := client.Users.GetUser(ctx, connect.NewRequest(&adminv1.GetUserRequest{
		ProjectId: projectID, UserId: ref,
	}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

// dialProject resolves the client and the --project reference in one step.
func dialProject(cmd *cobra.Command, opts *clientOpts, project string) (*cli.Client, *adminv1.Project, error) {
	client, err := opts.dial()
	if err != nil {
		return nil, nil, err
	}
	p, err := resolveProject(cmd.Context(), client, project)
	if err != nil {
		return nil, nil, err
	}
	return client, p, nil
}

func newUserListCmd(opts *clientOpts, project *string) *cobra.Command {
	var query string
	var pageSize int32
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List users, newest first",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, p, err := dialProject(cmd, opts, *project)
			if err != nil {
				return err
			}
			resp, err := client.Users.ListUsers(cmd.Context(), connect.NewRequest(&adminv1.ListUsersRequest{
				ProjectId: p.Id, Query: query, PageSize: pageSize,
			}))
			if err != nil {
				return err
			}
			if opts.json {
				return printJSON(cmd, resp.Msg)
			}
			rows := make([][]string, 0, len(resp.Msg.Users))
			for _, u := range resp.Msg.Users {
				rows = append(rows, []string{u.Id, u.Email, fmtBool(u.EmailVerified),
					fmtBool(u.Disabled), strings.Join(u.Providers, ","), fmtTime(u.CreateTime)})
			}
			if err := table(cmd, []string{"ID", "EMAIL", "VERIFIED", "DISABLED", "PROVIDERS", "CREATED"}, rows); err != nil {
				return err
			}
			if resp.Msg.NextPageToken != "" {
				fmt.Printf("(%d of %d users; narrow with --query)\n",
					len(resp.Msg.Users), resp.Msg.TotalSize)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "substring filter on email and display name")
	cmd.Flags().Int32Var(&pageSize, "page-size", 0, "users per page (server default 50, max 200)")
	return cmd
}

func newUserGetCmd(opts *clientOpts, project *string) *cobra.Command {
	return &cobra.Command{
		Use:   "get <id|email>",
		Short: "Show one user with identities and active sessions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, p, err := dialProject(cmd, opts, *project)
			if err != nil {
				return err
			}
			resp, err := resolveUser(cmd.Context(), client, p.Id, args[0])
			if err != nil {
				return err
			}
			if opts.json {
				return printJSON(cmd, resp)
			}
			u := resp.User
			fmt.Printf("%s\n", u.Email)
			fmt.Printf("  id:            %s\n", u.Id)
			fmt.Printf("  display name:  %s\n", u.DisplayName)
			fmt.Printf("  verified:      %s\n", fmtBool(u.EmailVerified))
			fmt.Printf("  disabled:      %s\n", fmtBool(u.Disabled))
			fmt.Printf("  providers:     %s\n", strings.Join(u.Providers, ", "))
			fmt.Printf("  created:       %s\n", fmtTime(u.CreateTime))
			fmt.Printf("  last login:    %s\n", fmtTime(u.LastLoginTime))
			if u.CustomClaims != "" {
				fmt.Printf("  custom claims: %s\n", u.CustomClaims)
			}
			fmt.Printf("  sessions:      %d active\n", len(resp.Sessions))
			return nil
		},
	}
}

func newUserCreateCmd(opts *clientOpts, project *string) *cobra.Command {
	var displayName, password string
	var verified, invite bool
	cmd := &cobra.Command{
		Use:   "create <email>",
		Short: "Create a user (with a password, or with an invite email)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, p, err := dialProject(cmd, opts, *project)
			if err != nil {
				return err
			}
			resp, err := client.Users.CreateUser(cmd.Context(), connect.NewRequest(&adminv1.CreateUserRequest{
				ProjectId: p.Id, Email: args[0], DisplayName: displayName,
				Password: password, EmailVerified: verified, SendInvite: invite,
			}))
			if err != nil {
				return err
			}
			if opts.json {
				return printJSON(cmd, resp.Msg)
			}
			fmt.Printf("created user %s (id %s)\n", resp.Msg.User.Email, resp.Msg.User.Id)
			return nil
		},
	}
	cmd.Flags().StringVar(&displayName, "display-name", "", "display name")
	cmd.Flags().StringVar(&password, "password", "", "initial password (omit with --invite to let the user choose one)")
	cmd.Flags().BoolVar(&verified, "verified", false, "mark the email address as already verified")
	cmd.Flags().BoolVar(&invite, "invite", false, "send a set-password invite email")
	return cmd
}

func newUserInviteCmd(opts *clientOpts, project *string) *cobra.Command {
	var displayName string
	cmd := &cobra.Command{
		Use:   "invite <email>",
		Short: "Create a user and email them a set-password invite",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, p, err := dialProject(cmd, opts, *project)
			if err != nil {
				return err
			}
			resp, err := client.Users.CreateUser(cmd.Context(), connect.NewRequest(&adminv1.CreateUserRequest{
				ProjectId: p.Id, Email: args[0], DisplayName: displayName, SendInvite: true,
			}))
			if err != nil {
				return err
			}
			if opts.json {
				return printJSON(cmd, resp.Msg)
			}
			fmt.Printf("invited %s (id %s)\n", resp.Msg.User.Email, resp.Msg.User.Id)
			return nil
		},
	}
	cmd.Flags().StringVar(&displayName, "display-name", "", "display name")
	return cmd
}

func newUserDisableCmd(opts *clientOpts, project *string) *cobra.Command {
	return &cobra.Command{
		Use:   "disable <id|email>",
		Short: "Block sign-in and revoke the user's sessions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, p, err := dialProject(cmd, opts, *project)
			if err != nil {
				return err
			}
			u, err := resolveUser(cmd.Context(), client, p.Id, args[0])
			if err != nil {
				return err
			}
			resp, err := client.Users.DisableUser(cmd.Context(), connect.NewRequest(&adminv1.DisableUserRequest{
				ProjectId: p.Id, UserId: u.User.Id,
			}))
			if err != nil {
				return err
			}
			if opts.json {
				return printJSON(cmd, resp.Msg)
			}
			fmt.Printf("disabled %s\n", resp.Msg.User.Email)
			return nil
		},
	}
}

func newUserEnableCmd(opts *clientOpts, project *string) *cobra.Command {
	return &cobra.Command{
		Use:   "enable <id|email>",
		Short: "Re-enable a disabled user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, p, err := dialProject(cmd, opts, *project)
			if err != nil {
				return err
			}
			u, err := resolveUser(cmd.Context(), client, p.Id, args[0])
			if err != nil {
				return err
			}
			resp, err := client.Users.EnableUser(cmd.Context(), connect.NewRequest(&adminv1.EnableUserRequest{
				ProjectId: p.Id, UserId: u.User.Id,
			}))
			if err != nil {
				return err
			}
			if opts.json {
				return printJSON(cmd, resp.Msg)
			}
			fmt.Printf("enabled %s\n", resp.Msg.User.Email)
			return nil
		},
	}
}

func newUserDeleteCmd(opts *clientOpts, project *string) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <id|email>",
		Short: "Permanently delete a user, their identities and sessions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, p, err := dialProject(cmd, opts, *project)
			if err != nil {
				return err
			}
			u, err := resolveUser(cmd.Context(), client, p.Id, args[0])
			if err != nil {
				return err
			}
			if err := confirm(cmd, yes, fmt.Sprintf("delete user %s permanently", u.User.Email)); err != nil {
				return err
			}
			if _, err := client.Users.DeleteUser(cmd.Context(), connect.NewRequest(&adminv1.DeleteUserRequest{
				ProjectId: p.Id, UserId: u.User.Id,
			})); err != nil {
				return err
			}
			fmt.Printf("deleted %s\n", u.User.Email)
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt")
	return cmd
}

func newUserClaimsCmd(opts *clientOpts, project *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claims",
		Short: "Manage a user's custom JWT claims",
	}
	cmd.AddCommand(newUserClaimsSetCmd(opts, project))
	return cmd
}

func newUserClaimsSetCmd(opts *clientOpts, project *string) *cobra.Command {
	return &cobra.Command{
		Use:   "set <id|email> <json>",
		Short: "Replace the user's custom claims with a JSON object ('{}' clears them)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var probe map[string]any
			if err := json.Unmarshal([]byte(args[1]), &probe); err != nil {
				return fmt.Errorf("claims must be a JSON object: %w", err)
			}
			client, p, err := dialProject(cmd, opts, *project)
			if err != nil {
				return err
			}
			u, err := resolveUser(cmd.Context(), client, p.Id, args[0])
			if err != nil {
				return err
			}
			resp, err := client.Users.UpdateUser(cmd.Context(), connect.NewRequest(&adminv1.UpdateUserRequest{
				ProjectId: p.Id, UserId: u.User.Id,
				User:       &adminv1.User{CustomClaims: args[1]},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"custom_claims"}},
			}))
			if err != nil {
				return err
			}
			if opts.json {
				return printJSON(cmd, resp.Msg)
			}
			fmt.Printf("claims of %s updated\n", resp.Msg.User.Email)
			return nil
		},
	}
}

func newUserSessionsCmd(opts *clientOpts, project *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sessions",
		Short: "Manage a user's device sessions",
	}
	cmd.AddCommand(newUserSessionsRevokeCmd(opts, project))
	return cmd
}

func newUserSessionsRevokeCmd(opts *clientOpts, project *string) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "revoke <id|email>",
		Short: "Revoke every session of the user (all devices sign out)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, p, err := dialProject(cmd, opts, *project)
			if err != nil {
				return err
			}
			u, err := resolveUser(cmd.Context(), client, p.Id, args[0])
			if err != nil {
				return err
			}
			if err := confirm(cmd, yes, fmt.Sprintf("revoke every session of %s", u.User.Email)); err != nil {
				return err
			}
			resp, err := client.Users.RevokeUserSessions(cmd.Context(), connect.NewRequest(&adminv1.RevokeUserSessionsRequest{
				ProjectId: p.Id, UserId: u.User.Id,
			}))
			if err != nil {
				return err
			}
			if opts.json {
				return printJSON(cmd, resp.Msg)
			}
			fmt.Printf("revoked %d sessions of %s\n", resp.Msg.RevokedCount, u.User.Email)
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt")
	return cmd
}
