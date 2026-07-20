package main

import (
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
)

func newInstanceCmd() *cobra.Command {
	var opts clientOpts
	cmd := &cobra.Command{
		Use:   "instance",
		Short: "Instance-wide settings of a moth server (remote)",
	}
	addClientFlags(cmd, &opts)
	cmd.AddCommand(newInstanceGetCmd(&opts), newInstanceSMTPCmd(&opts))
	return cmd
}

func newInstanceGetCmd(opts *clientOpts) *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Show base URL, version and effective SMTP settings",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := opts.dial()
			if err != nil {
				return err
			}
			resp, err := client.Settings.GetInstanceSettings(cmd.Context(),
				connect.NewRequest(&adminv1.GetInstanceSettingsRequest{}))
			if err != nil {
				return err
			}
			if opts.json {
				return printJSON(cmd, resp.Msg)
			}
			m := resp.Msg
			fmt.Printf("base url:  %s\n", m.BaseUrl)
			fmt.Printf("version:   %s\n", m.Version)
			fmt.Printf("smtp:      %s\n", smtpSourceLabel(m.SmtpSource))
			if s := m.Smtp; s.GetHost() != "" {
				fmt.Printf("  host: %s:%d  from: %s  user: %s  password: %s\n",
					s.Host, s.Port, s.From, s.Username, fmtBool(m.SmtpHasPassword))
			}
			return nil
		},
	}
}

func smtpSourceLabel(s adminv1.SmtpSource) string {
	switch s {
	case adminv1.SmtpSource_SMTP_SOURCE_NONE:
		return "none (emails are logged to the server console)"
	case adminv1.SmtpSource_SMTP_SOURCE_CONFIG:
		return "from the server config file"
	case adminv1.SmtpSource_SMTP_SOURCE_DATABASE:
		return "from the database (set via admin console or CLI)"
	default:
		return strings.TrimPrefix(s.String(), "SMTP_SOURCE_")
	}
}

func newInstanceSMTPCmd(opts *clientOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "smtp",
		Short: "Configure and test outgoing email",
	}
	cmd.AddCommand(newInstanceSMTPSetCmd(opts), newInstanceSMTPTestCmd(opts), newInstanceSMTPClearCmd(opts))
	return cmd
}

func newInstanceSMTPSetCmd(opts *clientOpts) *cobra.Command {
	var host, username, password, from string
	var port int32
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Store an SMTP configuration (takes precedence over the config file)",
		Long: `Set stores the SMTP relay configuration in the database. An empty
--password keeps the currently stored one; use 'moth instance smtp clear'
to drop the stored configuration and fall back to the config file.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := opts.dial()
			if err != nil {
				return err
			}
			resp, err := client.Settings.UpdateSmtpSettings(cmd.Context(),
				connect.NewRequest(&adminv1.UpdateSmtpSettingsRequest{
					Smtp: &adminv1.SmtpSettings{
						Host: host, Port: port, Username: username,
						Password: password, From: from,
					},
				}))
			if err != nil {
				return err
			}
			if opts.json {
				return printJSON(cmd, resp.Msg)
			}
			fmt.Printf("smtp settings stored (%s)\n", smtpSourceLabel(resp.Msg.SmtpSource))
			return nil
		},
	}
	cmd.Flags().StringVar(&host, "host", "", "SMTP host (required)")
	cmd.Flags().Int32Var(&port, "port", 587, "SMTP port")
	cmd.Flags().StringVar(&username, "username", "", "SMTP username")
	cmd.Flags().StringVar(&password, "password", "", "SMTP password (empty keeps the stored one)")
	cmd.Flags().StringVar(&from, "from", "", "sender address (required)")
	_ = cmd.MarkFlagRequired("host") // flag is registered just above
	_ = cmd.MarkFlagRequired("from") // flag is registered just above
	return cmd
}

func newInstanceSMTPClearCmd(opts *clientOpts) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Drop the stored SMTP configuration (falls back to the config file, then console)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := confirm(cmd, yes, "clear the stored SMTP configuration"); err != nil {
				return err
			}
			client, err := opts.dial()
			if err != nil {
				return err
			}
			resp, err := client.Settings.UpdateSmtpSettings(cmd.Context(),
				connect.NewRequest(&adminv1.UpdateSmtpSettingsRequest{Smtp: &adminv1.SmtpSettings{}}))
			if err != nil {
				return err
			}
			if opts.json {
				return printJSON(cmd, resp.Msg)
			}
			fmt.Printf("stored smtp settings cleared (%s)\n", smtpSourceLabel(resp.Msg.SmtpSource))
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt")
	return cmd
}

func newInstanceSMTPTestCmd(opts *clientOpts) *cobra.Command {
	var to string
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Send a probe email through the effective transport",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := opts.dial()
			if err != nil {
				return err
			}
			if _, err := client.Settings.SendTestEmail(cmd.Context(),
				connect.NewRequest(&adminv1.SendTestEmailRequest{To: to})); err != nil {
				return err
			}
			fmt.Printf("test email sent to %s\n", to)
			return nil
		},
	}
	cmd.Flags().StringVar(&to, "to", "", "recipient address (required)")
	_ = cmd.MarkFlagRequired("to") // flag is registered just above
	return cmd
}
