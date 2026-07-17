package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/internal/cli"
)

func newProjectCmd() *cobra.Command {
	var opts clientOpts
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage the projects of a moth server (remote)",
	}
	addClientFlags(cmd, &opts)
	cmd.AddCommand(
		newProjectCreateCmd(&opts),
		newProjectListCmd(&opts),
		newProjectGetCmd(&opts),
		newProjectUpdateCmd(&opts),
		newProjectDeleteCmd(&opts),
		newProjectKeysCmd(&opts),
		newProjectDumpCmd(&opts),
		newProjectApplyCmd(&opts),
		newProjectExportCmd(&opts),
		newProjectImportCmd(&opts),
	)
	return cmd
}

// resolveProject finds a project by slug or id. The not-found error carries
// connect.CodeNotFound so the process exits with the not-found code.
func resolveProject(ctx context.Context, client *cli.Client, ref string) (*adminv1.Project, error) {
	list, err := client.Projects.ListProjects(ctx, connect.NewRequest(&adminv1.ListProjectsRequest{}))
	if err != nil {
		return nil, err
	}
	for _, p := range list.Msg.Projects {
		if p.Slug == ref || p.Id == ref {
			return p, nil
		}
	}
	return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("project %q not found", ref))
}

func newProjectCreateCmd(opts *clientOpts) *cobra.Command {
	var slug string
	var showSecret bool
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := opts.dial()
			if err != nil {
				return err
			}
			resp, err := client.Projects.CreateProject(cmd.Context(),
				connect.NewRequest(&adminv1.CreateProjectRequest{Name: args[0], Slug: slug}))
			if err != nil {
				return err
			}
			msg := resp.Msg
			if !showSecret {
				msg.SecretKey = ""
			}
			if opts.json {
				return printJSON(cmd, msg)
			}
			p := msg.Project
			fmt.Printf("created project %q (slug %s, id %s)\npublishable key: %s\n",
				p.Name, p.Slug, p.Id, p.PublishableKey)
			if showSecret {
				fmt.Printf("secret key (shown exactly once): %s\n", msg.SecretKey)
			} else {
				fmt.Println(
					"secret key hidden (--show-secret to print it; it is shown exactly once — recover with 'moth project keys regenerate-secret --show-secret')")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&slug, "slug", "", "explicit slug (default: derived from the name)")
	cmd.Flags().BoolVar(&showSecret, "show-secret", false, "print the server-to-server secret key")
	return cmd
}

func newProjectListCmd(opts *clientOpts) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List projects",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := opts.dial()
			if err != nil {
				return err
			}
			resp, err := client.Projects.ListProjects(cmd.Context(),
				connect.NewRequest(&adminv1.ListProjectsRequest{}))
			if err != nil {
				return err
			}
			if opts.json {
				return printJSON(cmd, resp.Msg)
			}
			rows := make([][]string, 0, len(resp.Msg.Projects))
			for _, p := range resp.Msg.Projects {
				rows = append(rows, []string{p.Slug, p.Name, p.Id,
					fmt.Sprint(p.UserCount), fmtTime(p.CreateTime)})
			}
			return table(cmd, []string{"SLUG", "NAME", "ID", "USERS", "CREATED"}, rows)
		},
	}
}

func newProjectGetCmd(opts *clientOpts) *cobra.Command {
	return &cobra.Command{
		Use:   "get <slug|id>",
		Short: "Show one project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := opts.dial()
			if err != nil {
				return err
			}
			p, err := resolveProject(cmd.Context(), client, args[0])
			if err != nil {
				return err
			}
			if opts.json {
				return printJSON(cmd, p)
			}
			s := p.Settings
			fmt.Printf("%s (%s)\n", p.Name, p.Slug)
			fmt.Printf("  id:               %s\n", p.Id)
			fmt.Printf("  publishable key:  %s\n", p.PublishableKey)
			fmt.Printf("  users:            %d\n", p.UserCount)
			fmt.Printf("  created:          %s\n", fmtTime(p.CreateTime))
			fmt.Printf("  public signup:    %s\n", fmtBool(s.GetAllowPublicSignup()))
			fmt.Printf("  email verify:     %s\n", fmtBool(s.GetRequireEmailVerification()))
			fmt.Printf("  google sign-in:   %s\n", fmtBool(s.GetGoogle().GetEnabled()))
			fmt.Printf("  apple sign-in:    %s\n", fmtBool(s.GetApple().GetEnabled()))
			return nil
		},
	}
}

func newProjectUpdateCmd(opts *clientOpts) *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "update <slug|id>",
		Short: "Update a project (settings are edited declaratively: see 'moth project apply')",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return errors.New("nothing to update: pass --name (settings are edited with 'moth project apply')")
			}
			client, err := opts.dial()
			if err != nil {
				return err
			}
			p, err := resolveProject(cmd.Context(), client, args[0])
			if err != nil {
				return err
			}
			resp, err := client.Projects.UpdateProject(cmd.Context(),
				connect.NewRequest(&adminv1.UpdateProjectRequest{
					Id: p.Id, Name: name,
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
				}))
			if err != nil {
				return err
			}
			if opts.json {
				return printJSON(cmd, resp.Msg)
			}
			fmt.Printf("project %s renamed to %q\n", resp.Msg.Project.Slug, resp.Msg.Project.Name)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "new display name")
	return cmd
}

func newProjectDeleteCmd(opts *clientOpts) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <slug|id>",
		Short: "Delete a project and all its users, keys and tokens",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := opts.dial()
			if err != nil {
				return err
			}
			p, err := resolveProject(cmd.Context(), client, args[0])
			if err != nil {
				return err
			}
			action := fmt.Sprintf("delete project %q (%d users) permanently", p.Slug, p.UserCount)
			if err := confirm(cmd, yes, action); err != nil {
				return err
			}
			if _, err := client.Projects.DeleteProject(cmd.Context(),
				connect.NewRequest(&adminv1.DeleteProjectRequest{Id: p.Id})); err != nil {
				return err
			}
			fmt.Printf("deleted project %s\n", p.Slug)
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt")
	return cmd
}

func newProjectKeysCmd(opts *clientOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keys",
		Short: "Manage a project's signing and secret keys",
	}
	cmd.AddCommand(
		newProjectKeysShowCmd(opts),
		newProjectKeysRotateSigningCmd(opts),
		newProjectKeysResetSigningCmd(opts),
		newProjectKeysRegenerateSecretCmd(opts),
	)
	return cmd
}

func newProjectKeysRotateSigningCmd(opts *clientOpts) *cobra.Command {
	var graceSeconds int32
	cmd := &cobra.Command{
		Use:   "rotate <slug|id>",
		Short: "Rotate the signing key gracefully (old key stays in the JWKS during a grace period; no user is signed out)",
		Long: `Rotate mints a fresh ES256 signing key that signs new tokens from now
on, while the previous public key stays in the project JWKS for a grace
period so in-flight access tokens keep validating and no user is signed
out. Expired grace keys are pruned automatically.

Unlike 'moth project keys reset-signing', rotate never invalidates
existing sessions. Pass --grace-seconds to override the default grace
(the project's access-token TTL plus a clock-skew margin).`,
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
			resp, err := client.Projects.RotateSigningKey(cmd.Context(),
				connect.NewRequest(&adminv1.RotateSigningKeyRequest{
					ProjectId: p.Id, GraceSeconds: graceSeconds,
				}))
			if err != nil {
				return err
			}
			if opts.json {
				return printJSON(cmd, resp.Msg)
			}
			fmt.Printf("rotated signing key of %s (new kid %s)\n", p.Slug, resp.Msg.Key.GetKid())
			fmt.Printf("previous key stays in the JWKS until %s\n", fmtTime(resp.Msg.GraceExpireTime))
			return nil
		},
	}
	cmd.Flags().Int32Var(&graceSeconds, "grace-seconds", 0,
		"seconds the previous key stays in the JWKS (0 = access-token TTL + clock skew)")
	return cmd
}

func newProjectKeysShowCmd(opts *clientOpts) *cobra.Command {
	return &cobra.Command{
		Use:   "show <slug|id>",
		Short: "Show the active token-signing key and JWKS/issuer values",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := opts.dial()
			if err != nil {
				return err
			}
			p, err := resolveProject(cmd.Context(), client, args[0])
			if err != nil {
				return err
			}
			resp, err := client.Projects.GetSigningKey(cmd.Context(),
				connect.NewRequest(&adminv1.GetSigningKeyRequest{ProjectId: p.Id}))
			if err != nil {
				return err
			}
			if opts.json {
				return printJSON(cmd, resp.Msg)
			}
			fmt.Printf("kid:       %s\n", resp.Msg.Key.GetKid())
			fmt.Printf("algorithm: %s\n", resp.Msg.Key.GetAlgorithm())
			fmt.Printf("created:   %s\n", fmtTime(resp.Msg.Key.GetCreateTime()))
			fmt.Printf("jwks url:  %s\n", resp.Msg.JwksUrl)
			fmt.Printf("issuer:    %s\n", resp.Msg.Issuer)
			fmt.Printf("audience:  %s\n", resp.Msg.Audience)
			fmt.Printf("%s", resp.Msg.Key.GetPublicKeyPem())
			return nil
		},
	}
}

func newProjectKeysResetSigningCmd(opts *clientOpts) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "reset-signing <slug|id>",
		Short: "Replace the signing keypair (invalidates every issued token; all users must sign in again)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := opts.dial()
			if err != nil {
				return err
			}
			p, err := resolveProject(cmd.Context(), client, args[0])
			if err != nil {
				return err
			}
			action := fmt.Sprintf("reset the signing key of %q — every access token dies and all users must sign in again", p.Slug)
			if err := confirm(cmd, yes, action); err != nil {
				return err
			}
			resp, err := client.Projects.ResetSigningKey(cmd.Context(),
				connect.NewRequest(&adminv1.ResetSigningKeyRequest{ProjectId: p.Id}))
			if err != nil {
				return err
			}
			if opts.json {
				return printJSON(cmd, resp.Msg)
			}
			fmt.Printf("new signing key installed (kid %s)\n", resp.Msg.Key.GetKid())
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt")
	return cmd
}

func newProjectKeysRegenerateSecretCmd(opts *clientOpts) *cobra.Command {
	var yes, showSecret bool
	cmd := &cobra.Command{
		Use:   "regenerate-secret <slug|id>",
		Short: "Replace the server-to-server secret key (the old one stops working immediately)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !showSecret {
				// Refuse before rotating: the new key is returned exactly
				// once, so a run that hid it (say, in retained CI logs by
				// accident the other way round) would kill the old key with
				// no way to ever read the new one.
				return errors.New(
					"the new secret key is printed exactly once and can never be retrieved again; re-run with --show-secret to acknowledge printing it")
			}
			client, err := opts.dial()
			if err != nil {
				return err
			}
			p, err := resolveProject(cmd.Context(), client, args[0])
			if err != nil {
				return err
			}
			action := fmt.Sprintf("regenerate the secret key of %q — the current key stops working immediately", p.Slug)
			if err := confirm(cmd, yes, action); err != nil {
				return err
			}
			resp, err := client.Projects.RegenerateSecretKey(cmd.Context(),
				connect.NewRequest(&adminv1.RegenerateSecretKeyRequest{ProjectId: p.Id}))
			if err != nil {
				return err
			}
			if opts.json {
				return printJSON(cmd, resp.Msg)
			}
			fmt.Printf("new secret key (shown exactly once): %s\n", resp.Msg.SecretKey)
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt")
	cmd.Flags().BoolVar(&showSecret, "show-secret", false,
		"print the new secret key (required: it is shown exactly once)")
	return cmd
}

func newProjectDumpCmd(opts *clientOpts) *cobra.Command {
	return &cobra.Command{
		Use:   "dump [slug|id]",
		Short: "Emit a project's desired-state YAML (the 'moth project apply' input)",
		Long: `Dump writes the project's full desired state — name, slug, settings and
theme — as YAML on stdout, the exact document 'moth project apply -f'
consumes. Write-only provider secrets never appear (only their has_*
presence flags); applying a dump keeps the stored secrets.

The slug is optional when the server hosts exactly one project.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := opts.dial()
			if err != nil {
				return err
			}
			var p *adminv1.Project
			if len(args) == 1 {
				if p, err = resolveProject(cmd.Context(), client, args[0]); err != nil {
					return err
				}
			} else {
				list, err := client.Projects.ListProjects(cmd.Context(),
					connect.NewRequest(&adminv1.ListProjectsRequest{}))
				if err != nil {
					return err
				}
				if n := len(list.Msg.Projects); n != 1 {
					return fmt.Errorf("the server hosts %d projects; pass the slug of the one to dump", n)
				}
				p = list.Msg.Projects[0]
			}
			theme, err := client.Themes.GetTheme(cmd.Context(),
				connect.NewRequest(&adminv1.GetThemeRequest{ProjectId: p.Id}))
			if err != nil {
				return err
			}
			spec := &adminv1.ProjectSpec{Name: p.Name, Slug: p.Slug, Settings: p.Settings}
			if !theme.Msg.IsDefault {
				spec.Theme = theme.Msg.Theme
			}
			ents, err := client.Entitlements.ListEntitlements(cmd.Context(),
				connect.NewRequest(&adminv1.ListEntitlementsRequest{ProjectId: p.Id}))
			if err != nil {
				return err
			}
			prods, err := client.Products.ListProducts(cmd.Context(),
				connect.NewRequest(&adminv1.ListProductsRequest{ProjectId: p.Id}))
			if err != nil {
				return err
			}
			spec.Monetization = cli.MonetizationSpecFromCatalog(ents.Msg.Entitlements, prods.Msg.Products)
			data, err := cli.SpecToYAML(spec)
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(data)
			return err
		},
	}
}

func newProjectApplyCmd(opts *clientOpts) *cobra.Command {
	var file string
	var yes bool
	cmd := &cobra.Command{
		Use:   "apply -f <spec.yaml>",
		Short: "Create or update a project to match a desired-state YAML (idempotent)",
		Long: `Apply reads a ProjectSpec YAML (see 'moth project dump'), diffs it
against the live project identified by its slug, and applies only what
differs: it creates the project when the slug is free, updates the name
and settings otherwise, and installs (or resets) the theme. Running the
same spec twice reports zero changes.

Unset numeric settings, an empty timezone, absent redirect_schemes and
redirect_origins lists and absent google/apple sections keep the
server's current values,
so partial specs keep unrelated fields untouched. Booleans are the
exception: proto3 cannot distinguish an omitted boolean from false, so a
partial spec that omits e.g. require_email_verification applies it as
false — the plan lists every settings field it is about to change; start
from 'moth project dump' to be safe. Write-only provider secrets present
in the spec are (re)written on every apply.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			data, err := os.ReadFile(file)
			if err != nil {
				return err
			}
			spec, err := cli.SpecFromYAML(data)
			if err != nil {
				return err
			}
			client, err := opts.dial()
			if err != nil {
				return err
			}
			return runApply(cmd, opts, client, spec, yes)
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", "", "spec YAML file (required)")
	cmd.Flags().BoolVar(&yes, "yes", false, "apply without the confirmation prompt")
	_ = cmd.MarkFlagRequired("file") // flag is registered just above
	return cmd
}

// applyResult is the --json output of `moth project apply`.
type applyResult struct {
	Plan         cli.ApplyPlan        `json:"plan"`
	Monetization cli.MonetizationPlan `json:"monetization"`
	Applied      bool                 `json:"applied"`
}

func runApply(cmd *cobra.Command, opts *clientOpts, client *cli.Client, spec *adminv1.ProjectSpec, yes bool) error {
	ctx := cmd.Context()

	// Read the live state the spec's slug points at.
	var current *adminv1.Project
	list, err := client.Projects.ListProjects(ctx, connect.NewRequest(&adminv1.ListProjectsRequest{}))
	if err != nil {
		return err
	}
	for _, p := range list.Msg.Projects {
		if p.Slug == spec.Slug {
			current = p
			break
		}
	}
	var theme *adminv1.GetThemeResponse
	if current != nil {
		resp, err := client.Themes.GetTheme(ctx,
			connect.NewRequest(&adminv1.GetThemeRequest{ProjectId: current.Id}))
		if err != nil {
			return err
		}
		theme = resp.Msg
	}

	plan, sendSettings, err := cli.PlanApply(spec, current, theme)
	if err != nil {
		return err
	}

	// Monetization catalog diff (full desired state, keyed on identifiers).
	var curEnts []*adminv1.Entitlement
	var curProds []*adminv1.Product
	if current != nil {
		el, err := client.Entitlements.ListEntitlements(ctx,
			connect.NewRequest(&adminv1.ListEntitlementsRequest{ProjectId: current.Id}))
		if err != nil {
			return err
		}
		pl, err := client.Products.ListProducts(ctx,
			connect.NewRequest(&adminv1.ListProductsRequest{ProjectId: current.Id}))
		if err != nil {
			return err
		}
		curEnts, curProds = el.Msg.Entitlements, pl.Msg.Products
	}
	monPlan := cli.PlanMonetization(spec.Monetization, curEnts, curProds)

	if plan.Empty() && monPlan.Empty() {
		if opts.json {
			return printJSONValue(cmd, applyResult{Plan: plan, Monetization: monPlan})
		}
		fmt.Printf("project %s: no changes\n", spec.Slug)
		return nil
	}

	if !opts.json {
		fmt.Printf("project %s:\n", spec.Slug)
		for _, line := range plan.Summary() {
			fmt.Printf("  - %s\n", line)
		}
		for _, line := range monPlan.Summary() {
			fmt.Printf("  - %s\n", line)
		}
	}
	if err := confirm(cmd, yes, fmt.Sprintf("apply these changes to %q", spec.Slug)); err != nil {
		return err
	}

	if plan.Create {
		resp, err := client.Projects.CreateProject(ctx,
			connect.NewRequest(&adminv1.CreateProjectRequest{Name: spec.Name, Slug: spec.Slug}))
		if err != nil {
			return err
		}
		current = resp.Msg.Project
		if !opts.json {
			// The secret key is not printed here (apply may run in CI logs);
			// regenerate one when needed.
			fmt.Printf("created project %s (publishable key %s; secret key not shown — use 'moth project keys regenerate-secret')\n",
				current.Slug, current.PublishableKey)
		}
	}

	var paths []string
	if plan.UpdateName || plan.Create {
		paths = append(paths, "name")
	}
	if plan.UpdateSettings && sendSettings != nil {
		paths = append(paths, "settings")
	}
	// After a create the name is already right, but settings may need the
	// spec's values; skip the update entirely when nothing is in the mask
	// beyond what create covered.
	if (plan.UpdateSettings && sendSettings != nil) || plan.UpdateName {
		req := &adminv1.UpdateProjectRequest{
			Id:         current.Id,
			Name:       spec.Name,
			UpdateMask: &fieldmaskpb.FieldMask{Paths: paths},
		}
		if plan.UpdateSettings && sendSettings != nil {
			req.Settings = sendSettings
		}
		if _, err := client.Projects.UpdateProject(ctx, connect.NewRequest(req)); err != nil {
			return err
		}
	}

	if plan.UpdateTheme {
		if _, err := client.Themes.UpdateTheme(ctx, connect.NewRequest(&adminv1.UpdateThemeRequest{
			ProjectId: current.Id, Theme: spec.Theme,
		})); err != nil {
			return err
		}
	}
	if plan.ResetTheme {
		if _, err := client.Themes.ResetTheme(ctx, connect.NewRequest(&adminv1.ResetThemeRequest{
			ProjectId: current.Id,
		})); err != nil {
			return err
		}
	}

	if !monPlan.Empty() {
		if err := applyMonetization(ctx, client, current.Id, spec.Monetization, monPlan); err != nil {
			return err
		}
	}

	if opts.json {
		return printJSONValue(cmd, applyResult{Plan: plan, Monetization: monPlan, Applied: true})
	}
	fmt.Printf("project %s: applied\n", spec.Slug)
	return nil
}

// applyMonetization reconciles the project's catalog to the spec: it creates and
// updates entitlements, then products (resolving grant identifiers to entitlement
// ids), then deletes what the spec dropped — entitlements last so a product that
// still references one is updated off it before the delete cascades. Keyed on
// identifiers, so a re-applied dump makes no calls.
//
// Scope boundary (plan/12 "one catalog, three faces"): `moth project apply`
// drives the same tier/offering *definitions* as the admin screen and
// `moth setup billing`, but it only reconciles moth's own catalog — it does NOT
// push to App Store Connect / Google Play. A price changed via `apply` updates
// the desired state (and shows as drift in the admin status panel) but reaches
// the stores only when that catalog is pushed with `moth setup billing` or the
// admin "Sync store catalog" action. apply defines; setup/admin push.
func applyMonetization(ctx context.Context, client *cli.Client, projectID string, spec *adminv1.MonetizationSpec, plan cli.MonetizationPlan) error {
	// Current catalog, indexed by identifier.
	el, err := client.Entitlements.ListEntitlements(ctx,
		connect.NewRequest(&adminv1.ListEntitlementsRequest{ProjectId: projectID}))
	if err != nil {
		return err
	}
	entID := map[string]string{} // identifier -> server id
	for _, e := range el.Msg.Entitlements {
		entID[e.Identifier] = e.Id
	}
	pl, err := client.Products.ListProducts(ctx,
		connect.NewRequest(&adminv1.ListProductsRequest{ProjectId: projectID}))
	if err != nil {
		return err
	}
	prodID := map[string]string{} // identifier -> server id
	for _, p := range pl.Msg.Products {
		prodID[p.Identifier] = p.Id
	}
	specEnt := map[string]*adminv1.EntitlementSpec{}
	for _, e := range spec.GetEntitlements() {
		specEnt[e.Identifier] = e
	}
	specProd := map[string]*adminv1.ProductSpec{}
	for _, p := range spec.GetProducts() {
		specProd[p.Identifier] = p
	}

	// Entitlements: create then update (products may reference the new ones).
	for _, ident := range plan.CreateEntitlements {
		e := specEnt[ident]
		resp, err := client.Entitlements.CreateEntitlement(ctx, connect.NewRequest(&adminv1.CreateEntitlementRequest{
			ProjectId: projectID, Identifier: ident, DisplayName: e.GetDisplayName(),
		}))
		if err != nil {
			return err
		}
		entID[ident] = resp.Msg.Entitlement.Id
	}
	for _, ident := range plan.UpdateEntitlements {
		e := specEnt[ident]
		if _, err := client.Entitlements.UpdateEntitlement(ctx, connect.NewRequest(&adminv1.UpdateEntitlementRequest{
			ProjectId: projectID, Id: entID[ident], DisplayName: e.GetDisplayName(),
		})); err != nil {
			return err
		}
	}

	// Products: create/update, resolving grant identifiers to entitlement ids.
	resolveGrants := func(idents []string) ([]string, error) {
		ids := make([]string, 0, len(idents))
		for _, ident := range idents {
			id, ok := entID[ident]
			if !ok {
				return nil, fmt.Errorf("product grants unknown entitlement %q", ident)
			}
			ids = append(ids, id)
		}
		return ids, nil
	}
	for _, ident := range plan.CreateProducts {
		p := specProd[ident]
		grants, err := resolveGrants(p.GetEntitlements())
		if err != nil {
			return err
		}
		if _, err := client.Products.CreateProduct(ctx, connect.NewRequest(&adminv1.CreateProductRequest{
			ProjectId: projectID, Product: productFromSpec(p, grants),
		})); err != nil {
			return err
		}
	}
	for _, ident := range plan.UpdateProducts {
		p := specProd[ident]
		grants, err := resolveGrants(p.GetEntitlements())
		if err != nil {
			return err
		}
		if _, err := client.Products.UpdateProduct(ctx, connect.NewRequest(&adminv1.UpdateProductRequest{
			ProjectId: projectID, Id: prodID[ident], Product: productFromSpec(p, grants),
		})); err != nil {
			return err
		}
	}
	for _, ident := range plan.DeleteProducts {
		if _, err := client.Products.DeleteProduct(ctx, connect.NewRequest(&adminv1.DeleteProductRequest{
			ProjectId: projectID, Id: prodID[ident],
		})); err != nil {
			return err
		}
	}

	// Entitlements dropped from the spec: delete last.
	for _, ident := range plan.DeleteEntitlements {
		if _, err := client.Entitlements.DeleteEntitlement(ctx, connect.NewRequest(&adminv1.DeleteEntitlementRequest{
			ProjectId: projectID, Id: entID[ident],
		})); err != nil {
			return err
		}
	}
	return nil
}

// productFromSpec builds a Product message from a ProductSpec and resolved
// entitlement ids.
func productFromSpec(p *adminv1.ProductSpec, entitlementIDs []string) *adminv1.Product {
	return &adminv1.Product{
		Identifier:             p.GetIdentifier(),
		DisplayName:            p.GetDisplayName(),
		AppleProductId:         p.GetAppleProductId(),
		GoogleProductId:        p.GetGoogleProductId(),
		BillingPeriod:          p.GetBillingPeriod(),
		PriceAmountMicros:      p.GetPriceAmountMicros(),
		Currency:               p.GetCurrency(),
		TrialPeriod:            p.GetTrialPeriod(),
		IntroPriceAmountMicros: p.GetIntroPriceAmountMicros(),
		IntroPeriod:            p.GetIntroPeriod(),
		Offering:               p.GetOffering(),
		SortOrder:              p.GetSortOrder(),
		EntitlementIds:         entitlementIDs,
	}
}

// printJSONValue writes a plain Go value (not a proto message) as JSON.
func printJSONValue(cmd *cobra.Command, v any) error {
	data, err := jsonMarshalIndent(v)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), strings.TrimRight(string(data), "\n"))
	return err
}
