package main

import (
	"fmt"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
)

func newStatsCmd() *cobra.Command {
	var opts clientOpts
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Project analytics (remote)",
	}
	addClientFlags(cmd, &opts)
	cmd.AddCommand(newStatsGetCmd(&opts))
	return cmd
}

func newStatsGetCmd(opts *clientOpts) *cobra.Command {
	var project, from, to string
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Show a project's stat tiles and breakdowns",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, p, err := dialProject(cmd, opts, project)
			if err != nil {
				return err
			}
			if from == "" || to == "" {
				defFrom, defTo := dateFlagDefaults(30)
				if from == "" {
					from = defFrom
				}
				if to == "" {
					to = defTo
				}
			}
			resp, err := client.Analytics.GetStats(cmd.Context(), connect.NewRequest(&adminv1.GetStatsRequest{
				ProjectId: p.Id, FromDate: from, ToDate: to,
			}))
			if err != nil {
				return err
			}
			if opts.json {
				return printJSON(cmd, resp.Msg)
			}
			t := resp.Msg.Tiles
			fmt.Printf("%s — %s to %s\n", p.Slug, from, to)
			fmt.Printf("  total users:        %d\n", t.GetTotalUsers())
			fmt.Printf("  new users (7d):     %d (previous 7d: %d)\n",
				t.GetNewUsers_7D(), t.GetNewUsersPrevious_7D())
			if t.GetLatestDauDate() != "" {
				fmt.Printf("  latest DAU:         %d (%s)\n", t.GetLatestDau(), t.GetLatestDauDate())
			}
			fmt.Printf("  logins (7d):        %d (%d failed, %.0f%% success)\n",
				t.GetLogins_7D(), t.GetLoginFailures_7D(), t.GetLoginSuccessRate_7D()*100)
			pr := resp.Msg.Providers
			fmt.Printf("  logins by provider: password %d, google %d, apple %d\n",
				pr.GetPassword(), pr.GetGoogle(), pr.GetApple())
			pl := resp.Msg.Platforms
			fmt.Printf("  logins by platform: ios %d, android %d, web %d, other %d\n",
				pl.GetIos(), pl.GetAndroid(), pl.GetWeb(), pl.GetOther())
			return nil
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "project slug or id (required)")
	cmd.Flags().StringVar(&from, "from", "", "first day, YYYY-MM-DD (default: 30 days ago)")
	cmd.Flags().StringVar(&to, "to", "", "last day, YYYY-MM-DD (default: today)")
	_ = cmd.MarkFlagRequired("project") // flag is registered just above
	return cmd
}
