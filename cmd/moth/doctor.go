package main

import (
	"github.com/spf13/cobra"

	"github.com/aloisdeniel/moth/internal/setup"
)

func newDoctorCmd() *cobra.Command {
	var opts clientOpts
	d := &setup.Doctor{}
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Health-check a moth instance (and optionally one project's provider config)",
		Long: `Runs the support checklist for "login stopped working": admin API
reachability, base-URL/TLS sanity, health and pub endpoints, SMTP
configuration (with a real test send via --smtp-to), and — with --project
— the project's JWKS plus its Google/Apple provider configuration,
verified against the providers' live endpoints.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, baseURL, err := opts.dialURL()
			if err != nil {
				return err
			}
			d.BaseURL = baseURL
			d.Session = client.Sessions
			d.Settings = client.Settings
			d.Projects = client.Projects
			rep, err := d.Run(cmd.Context())
			if err != nil {
				return err
			}
			return printReport(cmd, rep, opts.json)
		},
	}
	addClientFlags(cmd, &opts)
	cmd.Flags().StringVar(&d.Slug, "project", "", "project slug to check provider config for")
	cmd.Flags().StringVar(&d.SMTPTestTo, "smtp-to", "", "send a real test email to this address")
	cmd.Flags().StringVar(&d.AppleKeyPath, "apple-key", "",
		"path to the project's Sign in with Apple .p8, enabling the Apple token-endpoint dry-run")
	return cmd
}
