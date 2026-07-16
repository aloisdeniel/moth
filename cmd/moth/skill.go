package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/internal/cli"
	"github.com/aloisdeniel/moth/internal/skill"
)

func newSkillCmd() *cobra.Command {
	var opts clientOpts
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Agent skill teaching coding agents to integrate and administer moth",
	}
	addClientFlags(cmd, &opts)
	cmd.AddCommand(newSkillExportCmd(&opts))
	return cmd
}

func newSkillExportCmd(opts *clientOpts) *cobra.Command {
	var (
		project string
		dir     string
		format  string
	)
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Write the skill directory (SKILL.md + references/)",
		Long: `Write the embedded moth agent skill to a directory: a SKILL.md with
name/description frontmatter plus references/ teaching an agent to
integrate the moth_auth Flutter SDK into an app and to administer a moth
server through this CLI.

With --project (and a configured context, see 'moth login') every snippet
is interpolated with that project's real values — endpoint, publishable
key, JWKS URL, enabled providers — the agent equivalent of the project's
setup-instructions page. Without --project the skill carries documented
placeholders instead and no server is contacted.

--format claude (default) follows Claude Code conventions; --format
generic writes plain markdown (README.md, no frontmatter) for other agent
frameworks. Exports are idempotent: re-running overwrites the files in
place, so regenerating after a config change is safe.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			f, err := skill.ParseFormat(format)
			if err != nil {
				return err
			}
			values := skill.Placeholders()
			if project != "" {
				values, err = fetchSkillValues(cmd.Context(), opts, project, cmd.ErrOrStderr())
				if err != nil {
					return err
				}
			}
			files, err := skill.Export(dir, f, values)
			if err != nil {
				return err
			}
			if opts.json {
				return printJSONValue(cmd, map[string]any{
					"dir":          dir,
					"files":        files,
					"format":       string(f),
					"interpolated": values.Interpolated,
				})
			}
			out := cmd.OutOrStdout()
			for _, rel := range files {
				if _, err := fmt.Fprintln(out, filepath.Join(dir, filepath.FromSlash(rel))); err != nil {
					return err
				}
			}
			if !values.Interpolated {
				_, err := fmt.Fprintln(out,
					"exported with placeholders; re-run with --project <slug> to interpolate real values")
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&project, "project", "",
		"project slug (or id) to interpolate real values for; requires a configured context")
	cmd.Flags().StringVar(&dir, "dir", filepath.Join(".claude", "skills", "moth"),
		"directory to write the skill into")
	cmd.Flags().StringVar(&format, "format", string(skill.FormatClaude),
		"skill flavor: claude (Claude Code conventions) or generic (plain markdown)")
	return cmd
}

// fetchSkillValues assembles the interpolation values for one project from
// the same admin RPCs the SPA's setup page uses, plus the public pub
// listing for the served SDK version.
func fetchSkillValues(ctx context.Context, opts *clientOpts, project string, warn io.Writer) (skill.Values, error) {
	cctx, err := opts.resolveContext()
	if err != nil {
		return skill.Values{}, err
	}
	client := cli.New(cctx.URL, cctx.Token)

	inst, err := client.Settings.GetInstanceSettings(ctx,
		connect.NewRequest(&adminv1.GetInstanceSettingsRequest{}))
	if err != nil {
		return skill.Values{}, err
	}
	p, err := resolveProject(ctx, client, project)
	if err != nil {
		return skill.Values{}, err
	}
	signing, err := client.Projects.GetSigningKey(ctx,
		connect.NewRequest(&adminv1.GetSigningKeyRequest{ProjectId: p.Id}))
	if err != nil {
		return skill.Values{}, err
	}

	// The served SDK version comes from the instance's public pub listing,
	// like the SPA setup page; a failure only degrades the pubspec snippet
	// to its placeholder, so it warns instead of aborting the export.
	sdkVersion, err := servedSDKVersion(ctx, cctx.URL)
	if err != nil {
		sdkVersion = skill.Placeholders().SDKVersion
		_, _ = fmt.Fprintf(warn, "warning: could not read the served SDK version (%v); "+
			"the pubspec snippet keeps the %s placeholder\n", err, sdkVersion)
	}

	v := skill.Values{
		Endpoint:       strings.TrimSuffix(inst.Msg.BaseUrl, "/"),
		ProjectName:    p.Name,
		Slug:           p.Slug,
		PublishableKey: p.PublishableKey,
		JWKSURL:        signing.Msg.JwksUrl,
		Issuer:         signing.Msg.Issuer,
		Audience:       signing.Msg.Audience,
		SDKVersion:     sdkVersion,
		Interpolated:   true,
	}
	if g := p.GetSettings().GetGoogle(); g != nil {
		v.Google = skill.GoogleValues{
			Enabled:         g.Enabled,
			WebClientID:     g.WebClientId,
			IOSClientID:     g.IosClientId,
			AndroidClientID: g.AndroidClientId,
		}
	}
	if a := p.GetSettings().GetApple(); a != nil {
		v.Apple = skill.AppleValues{
			Enabled:    a.Enabled,
			ServicesID: a.ServicesId,
			BundleIDs:  a.BundleIds,
		}
	}
	return v, nil
}

// servedSDKVersion reads the moth_auth version the instance serves from
// its pub listing and turns it into a pubspec constraint: pre-releases
// (dev builds) are pinned exactly — Dart version ranges never match
// pre-releases — and releases get a caret.
func servedSDKVersion(ctx context.Context, serverURL string) (string, error) {
	url := strings.TrimSuffix(serverURL, "/") + "/pub/api/packages/moth_auth"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	hc := &http.Client{Timeout: 30 * time.Second}
	resp, err := hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	var listing struct {
		Latest struct {
			Version string `json:"version"`
		} `json:"latest"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listing); err != nil {
		return "", fmt.Errorf("parse %s: %w", url, err)
	}
	v := listing.Latest.Version
	if v == "" {
		return "", fmt.Errorf("%s reports no latest version", url)
	}
	if strings.Contains(v, "-") {
		return v, nil
	}
	return "^" + v, nil
}
