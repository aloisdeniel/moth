package main

import (
	"fmt"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
)

func newTokenCmd() *cobra.Command {
	var opts clientOpts
	cmd := &cobra.Command{
		Use:   "token",
		Short: "Manage your personal access tokens (remote)",
	}
	addClientFlags(cmd, &opts)
	cmd.AddCommand(
		newTokenCreateCmd(&opts),
		newTokenListCmd(&opts),
		newTokenRevokeCmd(&opts),
	)
	return cmd
}

func newTokenCreateCmd(opts *clientOpts) *cobra.Command {
	var expiresInDays int32
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Mint a personal access token (the value is shown exactly once)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := opts.dial()
			if err != nil {
				return err
			}
			resp, err := client.Account.CreatePersonalAccessToken(cmd.Context(),
				connect.NewRequest(&adminv1.CreatePersonalAccessTokenRequest{
					Name: args[0], ExpiresInDays: expiresInDays,
				}))
			if err != nil {
				return err
			}
			if opts.json {
				return printJSON(cmd, resp.Msg)
			}
			fmt.Printf("token %q created (id %s, expires %s)\n%s\n",
				resp.Msg.Metadata.GetName(), resp.Msg.Metadata.GetId(),
				fmtTime(resp.Msg.Metadata.GetExpireTime()), resp.Msg.Token)
			return nil
		},
	}
	cmd.Flags().Int32Var(&expiresInDays, "expires-in-days", 0, "days until expiry (0: never expires)")
	return cmd
}

func newTokenListCmd(opts *clientOpts) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List your personal access tokens, newest first",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := opts.dial()
			if err != nil {
				return err
			}
			resp, err := client.Account.ListPersonalAccessTokens(cmd.Context(),
				connect.NewRequest(&adminv1.ListPersonalAccessTokensRequest{}))
			if err != nil {
				return err
			}
			if opts.json {
				return printJSON(cmd, resp.Msg)
			}
			rows := make([][]string, 0, len(resp.Msg.Tokens))
			for _, t := range resp.Msg.Tokens {
				rows = append(rows, []string{t.Id, t.Name, fmtTime(t.CreateTime),
					fmtTime(t.LastUsedTime), fmtTime(t.ExpireTime), fmtTime(t.RevokeTime)})
			}
			return table(cmd, []string{"ID", "NAME", "CREATED", "LAST USED", "EXPIRES", "REVOKED"}, rows)
		},
	}
}

func newTokenRevokeCmd(opts *clientOpts) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "revoke <id>",
		Short: "Revoke a personal access token (its next use fails immediately)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := confirm(cmd, yes, fmt.Sprintf("revoke token %s", args[0])); err != nil {
				return err
			}
			client, err := opts.dial()
			if err != nil {
				return err
			}
			if _, err := client.Account.RevokePersonalAccessToken(cmd.Context(),
				connect.NewRequest(&adminv1.RevokePersonalAccessTokenRequest{Id: args[0]})); err != nil {
				return err
			}
			fmt.Printf("revoked token %s\n", args[0])
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt")
	return cmd
}
