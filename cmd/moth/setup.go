package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/aloisdeniel/moth/internal/billing"
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
	cmd.AddCommand(newSetupGoogleCmd(&opts), newSetupAppleCmd(&opts), newSetupBillingCmd(&opts))
	return cmd
}

func newSetupBillingCmd(opts *clientOpts) *cobra.Command {
	s := &setup.BillingSetup{}
	var (
		ascIssuerID, ascKeyID, ascP8       string
		iapKeyID, iapIssuerID, iapP8       string
		saPath, pubsubSAPath, cloudProject string
		noConfirm                          bool
	)
	cmd := &cobra.Command{
		Use:   "billing",
		Short: "Configure store subscriptions for a project (credentials, catalog push, notifications, verify)",
		Long: `Configures a project's store monetization end to end: it stores the
Apple App Store Server API and Google Play Developer API credentials into
moth's encrypted billing config, pushes moth's product catalog into App
Store Connect and Google Play (automated where the store APIs allow it,
guided with exact values where they don't), wires the notification
endpoints, and verifies each store is reachable and authenticated.

The App Store Connect API key (--asc-*) drives the Apple catalog push and
is used in-process only, never stored. The In-App-Purchase key (--apple-
iap-*) and the Google service account are stored encrypted for the
milestone-11 billing engine. Idempotent: re-running diffs the current
store state and changes only what is needed.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, baseURL, err := opts.dialURL()
			if err != nil {
				return err
			}
			s.Projects = client.Projects
			s.Products = client.Products
			s.BillingCreds = client.BillingCreds
			s.BaseURL = baseURL
			s.Prompt = prompter(cmd, opts.json)
			s.Out = cmd.OutOrStdout()
			s.Yes = noConfirm

			if s.AppleBundleID != "" {
				if iapP8 != "" {
					raw, err := os.ReadFile(iapP8)
					if err != nil {
						return err
					}
					key, err := oidc.ParseP8(raw)
					if err != nil {
						return fmt.Errorf("%s: %w", iapP8, err)
					}
					s.AppleIAPKeyP8 = raw
					s.AppleIAPKey = key
				}
				s.AppleIAPKeyID = iapKeyID
				s.AppleIAPIssuerID = iapIssuerID
				if ascP8 != "" {
					asc, err := buildASC(s.Prompt, ascIssuerID, ascKeyID, ascP8)
					if err != nil {
						return err
					}
					s.ASC = asc
				}
			}
			if s.GooglePackageName != "" && saPath != "" {
				raw, err := os.ReadFile(saPath)
				if err != nil {
					return err
				}
				sa, err := billing.ParseServiceAccount(raw)
				if err != nil {
					return fmt.Errorf("%s: %w", saPath, err)
				}
				s.GoogleServiceAccountJSON = raw
				s.GoogleSA = sa
				if pubsubSAPath != "" {
					psRaw, err := os.ReadFile(pubsubSAPath)
					if err != nil {
						return err
					}
					psSA, err := billing.ParseServiceAccount(psRaw)
					if err != nil {
						return fmt.Errorf("%s: %w", pubsubSAPath, err)
					}
					s.GooglePubSubTokens = billing.NewGoogleTokenSource(psSA, "", nil, nil)
					s.GoogleCloudProject = cloudProject
				}
			}

			rep, err := s.Run(cmd.Context())
			if err != nil {
				return err
			}
			return printReport(cmd, rep, opts.json)
		},
	}
	cmd.Flags().StringVar(&s.Slug, "project", "", "project slug (required)")
	// Apple.
	cmd.Flags().StringVar(&s.AppleBundleID, "apple-bundle-id", "", "app bundle id (enables Apple; blank skips Apple)")
	cmd.Flags().StringVar(&s.AppleAppAppleID, "apple-app-apple-id", "", "the app's numeric App Store id")
	cmd.Flags().StringVar(&s.AppleAppID, "apple-app-id", "", "App Store Connect app resource id (for the catalog push)")
	cmd.Flags().StringVar(&iapKeyID, "apple-iap-key-id", "", "App Store Server API In-App-Purchase key id")
	cmd.Flags().StringVar(&iapIssuerID, "apple-iap-issuer-id", "", "App Store Server API issuer id")
	cmd.Flags().StringVar(&iapP8, "apple-iap-p8", "", "path to the App Store Server API In-App-Purchase .p8 (stored encrypted)")
	cmd.Flags().StringVar(&s.AppleNotificationSecret, "apple-notification-secret", "", "App Store Server Notifications shared secret (stored encrypted)")
	cmd.Flags().StringVar(&ascIssuerID, "asc-issuer-id", "", "App Store Connect API issuer id (catalog push; not stored)")
	cmd.Flags().StringVar(&ascKeyID, "asc-key-id", "", "App Store Connect API key id (catalog push; not stored)")
	cmd.Flags().StringVar(&ascP8, "asc-p8", "", "path to the App Store Connect API .p8 (catalog push; not stored)")
	// Google.
	cmd.Flags().StringVar(&s.GooglePackageName, "google-package-name", "", "Android application id (enables Google; blank skips Google)")
	cmd.Flags().StringVar(&saPath, "google-service-account", "", "path to the Play Developer API service-account JSON (stored encrypted)")
	cmd.Flags().StringVar(&s.GooglePubsubTopic, "google-pubsub-topic", "", "Cloud Pub/Sub topic for RTDN (projects/<p>/topics/<t> or a bare topic id)")
	cmd.Flags().StringVar(&s.GoogleRTDNSecret, "google-rtdn-secret", "", "RTDN push webhook shared secret (stored encrypted)")
	cmd.Flags().StringVar(&pubsubSAPath, "google-pubsub-service-account", "", "path to a pubsub-scoped SA JSON to create the RTDN topic/subscription (else guided)")
	cmd.Flags().StringVar(&cloudProject, "google-cloud-project", "", "GCP project the RTDN topic lives in (with --google-pubsub-service-account)")
	cmd.Flags().BoolVar(&noConfirm, "yes", false, "push to the live stores without the confirmation prompt")
	_ = cmd.MarkFlagRequired("project") // flag is registered just above
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
