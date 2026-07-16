package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/aloisdeniel/moth/internal/oidc"
	"github.com/aloisdeniel/moth/internal/setup"
)

// checklistColor enables ANSI colors when stdout is a terminal and
// NO_COLOR is unset.
func checklistColor() bool {
	return os.Getenv("NO_COLOR") == "" && term.IsTerminal(int(os.Stdout.Fd()))
}

// printReport renders a setup/doctor checklist (JSON in --json mode) and
// converts failures into a non-zero exit.
func printReport(cmd *cobra.Command, rep *setup.Report, asJSON bool) error {
	if asJSON {
		data, err := rep.JSON()
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), string(data)); err != nil {
			return err
		}
	} else {
		rep.Print(cmd.OutOrStdout(), checklistColor())
	}
	if rep.Failed() {
		return fmt.Errorf("%d check(s) failed", countFailed(rep))
	}
	return nil
}

func countFailed(rep *setup.Report) int {
	n := 0
	for _, c := range rep.Checks {
		if c.Status == setup.StatusFail {
			n++
		}
	}
	return n
}

// prompter builds the guided-flow prompter; in --json mode the transcript
// moves to stderr so stdout stays machine-readable.
func prompter(cmd *cobra.Command, asJSON bool) *setup.Prompter {
	out := cmd.OutOrStdout()
	if asJSON {
		out = cmd.ErrOrStderr()
	}
	return setup.NewPrompter(cmd.InOrStdin(), out)
}

func newSetupCmd() *cobra.Command {
	var opts clientOpts
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Configure sign-in providers for a project (automated where APIs exist, guided where they don't)",
	}
	addClientFlags(cmd, &opts)
	cmd.AddCommand(newSetupGoogleCmd(&opts), newSetupAppleCmd(&opts))
	return cmd
}

func newSetupGoogleCmd(opts *clientOpts) *cobra.Command {
	s := &setup.GoogleSetup{}
	cmd := &cobra.Command{
		Use:   "google",
		Short: "Configure Sign in with Google for a project",
		Long: `Configures Sign in with Google end to end: verifies the GCP project
(via gcloud when installed), computes Android signing fingerprints (via
keytool or pasted values), walks through creating the OAuth clients in the
Google console (client creation has no public API), writes the client IDs
into the moth project, and verifies each one against Google's endpoints.

Idempotent: re-running diffs the current configuration and only changes
what is needed.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, baseURL, err := opts.dialURL()
			if err != nil {
				return err
			}
			s.Projects = client.Projects
			s.BaseURL = baseURL
			s.Prompt = prompter(cmd, opts.json)
			s.Out = cmd.OutOrStdout()
			rep, err := s.Run(cmd.Context())
			if err != nil {
				return err
			}
			return printReport(cmd, rep, opts.json)
		},
	}
	cmd.Flags().StringVar(&s.Slug, "project", "", "project slug (required)")
	cmd.Flags().StringVar(&s.GCPProject, "gcp-project", "", "GCP project ID hosting the OAuth clients")
	cmd.Flags().StringVar(&s.IOSBundleID, "ios-bundle-id", "", "iOS app bundle ID (blank skips iOS)")
	cmd.Flags().StringVar(&s.AndroidPackage, "android-package", "", "Android application ID (blank skips Android)")
	cmd.Flags().StringVar(&s.AndroidSHA1, "android-sha1", "", "Android signing certificate SHA-1 fingerprint")
	cmd.Flags().StringVar(&s.AndroidSHA256, "android-sha256", "", "Android signing certificate SHA-256 fingerprint")
	cmd.Flags().StringVar(&s.Keystore, "keystore", "", "keystore path to compute the fingerprints with keytool")
	cmd.Flags().StringVar(&s.KeystorePass, "keystore-pass", "",
		"keystore password (prefer omitting it: the command prompts without echo, keeping the password out of shell history)")
	cmd.Flags().StringVar(&s.WebClientID, "web-client-id", "", "existing web OAuth client ID (skips the guided step)")
	cmd.Flags().StringVar(&s.IOSClientID, "ios-client-id", "", "existing iOS OAuth client ID (skips the guided step)")
	cmd.Flags().StringVar(&s.AndroidClientID, "android-client-id", "", "existing Android OAuth client ID (skips the guided step)")
	cmd.Flags().StringVar(&s.WebClientSecret, "web-client-secret", "", "web OAuth client secret (stored encrypted; blank keeps the stored one)")
	_ = cmd.MarkFlagRequired("project") // flag is registered just above
	return cmd
}

func newSetupAppleCmd(opts *clientOpts) *cobra.Command {
	s := &setup.AppleSetup{}
	var issuerID, keyID, p8Path string
	cmd := &cobra.Command{
		Use:   "apple",
		Short: "Configure Sign in with Apple for a project",
		Long: `Configures Sign in with Apple end to end, authenticated with an App
Store Connect API key (used in-process only, never stored): verifies or
creates the bundle ID, enables the Sign in with Apple capability, creates
the Sign in with Apple key (Apple serves the .p8 exactly once — it is
immediately stored, encrypted, in the moth project), walks through the
Services ID registration (which has no official API), and dry-runs a
minted client secret against Apple's token endpoint.

Idempotent: re-running diffs the current configuration and only changes
what is needed.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, baseURL, err := opts.dialURL()
			if err != nil {
				return err
			}
			s.Projects = client.Projects
			s.BaseURL = baseURL
			s.Prompt = prompter(cmd, opts.json)
			s.Out = cmd.OutOrStdout()
			asc, err := buildASC(s.Prompt, issuerID, keyID, p8Path)
			if err != nil {
				return err
			}
			s.ASC = asc
			rep, err := s.Run(cmd.Context())
			if err != nil {
				return err
			}
			return printReport(cmd, rep, opts.json)
		},
	}
	cmd.Flags().StringVar(&s.Slug, "project", "", "project slug (required)")
	cmd.Flags().StringVar(&issuerID, "issuer-id", "", "App Store Connect API issuer ID")
	cmd.Flags().StringVar(&keyID, "key-id", "", "App Store Connect API key ID")
	cmd.Flags().StringVar(&p8Path, "p8", "", "path to the App Store Connect API .p8 key")
	cmd.Flags().StringVar(&s.BundleID, "bundle-id", "", "app bundle ID")
	cmd.Flags().StringVar(&s.TeamID, "team-id", "", "Apple Developer Team ID")
	cmd.Flags().StringVar(&s.ServicesID, "services-id", "", "Services ID for the web-redirect flow")
	cmd.Flags().BoolVar(&s.RotateKey, "rotate-key", false, "create a fresh Sign in with Apple key even when one is stored")
	cmd.Flags().BoolVar(&s.UseUnofficialAPI, "unofficial-api", false,
		"reserved: drive the unofficial developer-portal API for Services IDs (evaluated and deliberately not implemented)")
	_ = cmd.MarkFlagRequired("project") // flag is registered just above
	return cmd
}

// buildASC assembles the App Store Connect client from flags, prompting
// for whatever is missing.
func buildASC(prompt *setup.Prompter, issuerID, keyID, p8Path string) (*setup.ASC, error) {
	var err error
	if issuerID == "" {
		if issuerID, err = prompt.Ask("App Store Connect issuer ID", setup.ValidateASCIssuerID); err != nil {
			return nil, err
		}
	} else if issuerID, err = setup.ValidateASCIssuerID(issuerID); err != nil {
		return nil, err
	}
	if keyID == "" {
		if keyID, err = prompt.Ask("App Store Connect API key ID", setup.ValidateAppleKeyID); err != nil {
			return nil, err
		}
	} else if keyID, err = setup.ValidateAppleKeyID(keyID); err != nil {
		return nil, err
	}
	if p8Path == "" {
		if p8Path, err = prompt.Ask("Path to the App Store Connect API .p8", nil); err != nil {
			return nil, err
		}
	}
	raw, err := os.ReadFile(p8Path)
	if err != nil {
		return nil, err
	}
	key, err := oidc.ParseP8(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", p8Path, err)
	}
	return &setup.ASC{IssuerID: issuerID, KeyID: keyID, Key: key}, nil
}
