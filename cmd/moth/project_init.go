package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/internal/cli"
	"github.com/aloisdeniel/moth/internal/setup"
)

// newProjectInitCmd is the CLI face of the milestone-22 wizard: the same
// ask-configure-defer flow as the admin SPA, as terminal prompts. It is
// composition, not a new write path — after the single confirmation it runs
// CreateProject plus the same per-domain admin RPCs `moth project apply`
// and the tabs call, then UpdateProfile with the answers themselves.
func newProjectInitCmd(opts *clientOpts) *cobra.Command {
	var specOut string
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a project through the guided wizard (interactive)",
		Long: `Init walks the milestone-22 wizard in the terminal: platforms, sign-in
(email/password defaults, Google/Apple with credentials entered in-flow or
deferred to 'moth setup google|apple'), monetization (the first entitlement
and its tiers; store credentials always deferred to 'moth setup billing')
and push notifications. Nothing is written until the final confirmation —
abandoning at any prompt creates nothing.

After creation the pk_/sk_ keys are printed exactly once, followed by the
derived setup checklist (whatever was deferred) and a ready-to-commit
'moth project apply' spec of what was just built.

Interactive only: with piped stdin it refuses and points at the scriptable
'moth project create' and 'moth project apply' instead.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Interactive only. The check is on the actual command input
			// stream so tests can inject a scripted reader; a real pipe
			// (an os.File that is not a terminal) is refused.
			if f, ok := cmd.InOrStdin().(*os.File); ok && !term.IsTerminal(int(f.Fd())) {
				return errors.New("moth project init is interactive and needs a terminal; script project creation with 'moth project create' and 'moth project apply' instead")
			}
			client, err := opts.dial()
			if err != nil {
				return err
			}
			answers, err := cli.RunInitWizard(setup.NewPrompter(cmd.InOrStdin(), cmd.OutOrStdout()))
			if err != nil {
				return err
			}
			return runProjectInit(cmd, client, answers, specOut)
		},
	}
	cmd.Flags().StringVar(&specOut, "spec-out", "",
		"write the emitted 'moth project apply' spec YAML to this file instead of printing it")
	return cmd
}

// runProjectInit executes the confirmed wizard answers: CreateProject, then
// the same per-domain admin RPCs the apply path uses (settings update,
// catalog creates), the push settings, and finally the profile. The pk_/sk_
// keys are printed immediately after CreateProject — before any follow-up
// write, so a later failure can never lose the once-shown secret key — and
// follow-up failures are collected and reported (with the tab or CLI command
// that finishes each), never aborting: the checklist and the apply spec are
// still printed, and the command exits non-zero if anything failed.
func runProjectInit(cmd *cobra.Command, client *cli.Client, answers *cli.InitAnswers, specOut string) error {
	ctx := cmd.Context()
	out := cmd.OutOrStdout()

	created, err := client.Projects.CreateProject(ctx,
		connect.NewRequest(&adminv1.CreateProjectRequest{Name: answers.Spec.Name, Slug: answers.Spec.Slug}))
	if err != nil {
		return err
	}
	proj := created.Msg.Project

	// The keys first — the sk_ key is shown exactly once, like the SPA
	// wizard's keys screen (recover later with 'moth project keys
	// regenerate-secret --show-secret'). Printed before any follow-up write:
	// the project exists from here on, so nothing below may abort the print.
	_, _ = fmt.Fprintf(out, "\ncreated project %q (slug %s, id %s)\n", proj.Name, proj.Slug, proj.Id)
	_, _ = fmt.Fprintf(out, "publishable key: %s\n", proj.PublishableKey)
	_, _ = fmt.Fprintf(out, "secret key (shown exactly once — store it now): %s\n", created.Msg.SecretKey)

	// Follow-up writes: failures are collected and reported, never fatal —
	// the gap stays honestly visible on the checklist below, and the exit
	// code says something went wrong.
	failed := 0
	fail := func(what, finish string, err error) {
		failed++
		_, _ = fmt.Fprintf(out, "\nwarning: could not configure %s: %v\n  the project itself is fine — finish it %s\n", what, err, finish)
	}

	// Settings: the answers merged over the fresh project's defaults, the
	// exact shape apply sends after a create.
	liveSettings := proj.Settings
	if answers.Spec.Settings != nil {
		updated, err := client.Projects.UpdateProject(ctx, connect.NewRequest(&adminv1.UpdateProjectRequest{
			Id:         proj.Id,
			Name:       proj.Name,
			Settings:   cli.MergeSettings(proj.Settings, answers.Spec.Settings),
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name", "settings"}},
		}))
		if err != nil {
			fail("the sign-in settings", "from the admin's Providers tab or 'moth setup google|apple'", err)
		} else {
			liveSettings = updated.Msg.Project.Settings
		}
	}

	// Catalog: the same reconcile the apply path runs, against the empty
	// fresh catalog (everything is a create).
	if mon := answers.Spec.Monetization; mon != nil {
		if plan := cli.PlanMonetization(mon, nil, nil); !plan.Empty() {
			if err := applyMonetization(ctx, client, proj.Id, mon, plan); err != nil {
				fail("the monetization catalog", "from the admin's Monetization tab or 'moth setup billing'", err)
			}
		}
	}

	if answers.Push != nil {
		if _, err := client.Push.UpdatePushSettings(ctx, connect.NewRequest(&adminv1.UpdatePushSettingsRequest{
			ProjectId: proj.Id, Settings: answers.Push,
		})); err != nil {
			fail("the push settings", "from the admin's Settings tab", err)
		}
	}

	// The profile last: the answers themselves, which the setup tab and the
	// derived checklist key off.
	if _, err := client.Profiles.UpdateProfile(ctx, connect.NewRequest(&adminv1.UpdateProfileRequest{
		ProjectId: proj.Id, Profile: answers.Profile,
	})); err != nil {
		fail("the setup profile", "from the admin's Settings tab", err)
	}

	// The derived checklist: what remains, straight from the server's
	// live-config probes.
	status, err := client.Profiles.GetProjectSetupStatus(ctx,
		connect.NewRequest(&adminv1.GetProjectSetupStatusRequest{ProjectId: proj.Id}))
	switch {
	case err != nil:
		fail("the setup checklist read", "on the admin's project overview", err)
	case len(status.Msg.Items) == 0:
		_, _ = fmt.Fprintln(out, "\nnothing outstanding — the project is fully configured")
	default:
		_, _ = fmt.Fprintln(out, "\nOutstanding setup:")
		for _, it := range status.Msg.Items {
			line := fmt.Sprintf("  - %s — %s", it.Title, it.Detail)
			switch {
			case it.CliCommand != "":
				line += fmt.Sprintf(" (run: %s)", it.CliCommand)
			case it.Tab != "":
				line += fmt.Sprintf(" (admin tab: %s)", it.Tab)
			}
			_, _ = fmt.Fprintln(out, line)
		}
	}

	// The ready-to-commit apply spec: rebuilt from the live state (like
	// 'moth project dump'), so re-applying it reports zero changes.
	if err := printApplySpec(ctx, client, out, proj, liveSettings, specOut); err != nil {
		fail("the apply spec", "later with 'moth project dump'", err)
	}

	if failed > 0 {
		return fmt.Errorf("the project was created and its keys printed above, but %d setup step(s) failed — finish them as noted", failed)
	}
	return nil
}

// specScopeNote is printed with every emitted spec so the output never
// overpromises what re-applying it covers.
const specScopeNote = "note: the spec covers name/settings/monetization; the setup profile and push settings are not part of apply specs."

// printApplySpec rebuilds the ready-to-commit apply spec from the live state
// and writes it to specOut (or prints it), with the scope note.
func printApplySpec(ctx context.Context, client *cli.Client, out io.Writer, proj *adminv1.Project, liveSettings *adminv1.ProjectSettings, specOut string) error {
	ents, err := client.Entitlements.ListEntitlements(ctx,
		connect.NewRequest(&adminv1.ListEntitlementsRequest{ProjectId: proj.Id}))
	if err != nil {
		return err
	}
	prods, err := client.Products.ListProducts(ctx,
		connect.NewRequest(&adminv1.ListProductsRequest{ProjectId: proj.Id}))
	if err != nil {
		return err
	}
	spec := &adminv1.ProjectSpec{
		Name:         proj.Name,
		Slug:         proj.Slug,
		Settings:     liveSettings,
		Monetization: cli.MonetizationSpecFromCatalog(ents.Msg.Entitlements, prods.Msg.Products),
	}
	data, err := cli.SpecToYAML(spec)
	if err != nil {
		return err
	}
	if specOut != "" {
		if err := os.WriteFile(specOut, data, 0o600); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(out, "\napply spec written to %s — commit it and re-apply with 'moth project apply -f %s'\n%s\n", specOut, specOut, specScopeNote)
		return nil
	}
	_, _ = fmt.Fprintf(out, "\n# Ready-to-commit desired state — save it and re-apply with 'moth project apply -f <file>':\n# %s\n%s", specScopeNote, data)
	return nil
}
