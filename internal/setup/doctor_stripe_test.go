package setup

import (
	"context"
	"testing"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
)

func TestDoctorStripeBilling(t *testing.T) {
	t.Run("configured stripe passes; probe needs the key in hand", func(t *testing.T) {
		srv := instanceDouble(t, nil)
		d, _ := newDoctor(t, srv, healthyProject())
		d.Slug = "demo"
		d.BillingCreds = &fakeBillingCreds{stripe: &adminv1.StripeBillingConfig{
			HasSecretKey: true, HasWebhookSecret: true, WebhookEndpointId: "we_1",
		}}
		d.Products = &fakeProducts{products: testProducts()}

		rep, err := d.Run(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{
			"project: Stripe billing credentials",
			"project: Stripe webhook secret",
			"project: Stripe webhook endpoint",
		} {
			if c := findCheck(t, rep, name); c.Status != StatusPass {
				t.Fatalf("%s = %s (%s)", name, c.Status, c.Detail)
			}
		}
		// moth never returns the stored key, so the remote probe degrades to a
		// warning until the operator supplies it in hand.
		if c := findCheck(t, rep, "project: Stripe API reachable"); c.Status != StatusWarn {
			t.Fatalf("probe without key = %+v", c)
		}
	})

	t.Run("in-hand key probes the Stripe API", func(t *testing.T) {
		srv := instanceDouble(t, nil)
		stripe := newStripeDouble(t)
		d, _ := newDoctor(t, srv, healthyProject())
		d.Slug = "demo"
		d.BillingCreds = &fakeBillingCreds{stripe: &adminv1.StripeBillingConfig{
			HasSecretKey: true, HasWebhookSecret: true,
		}}
		d.Products = &fakeProducts{products: testProducts()}
		d.StripeSecretKey = "sk_test_moth"
		d.StripeAPIBase = stripe.srv.URL

		rep, err := d.Run(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		// The double 404s the bogus price read: authenticated reachability.
		if c := findCheck(t, rep, "project: Stripe API reachable"); c.Status != StatusPass {
			t.Fatalf("probe = %+v", c)
		}
	})

	t.Run("missing webhook secret warns", func(t *testing.T) {
		srv := instanceDouble(t, nil)
		d, _ := newDoctor(t, srv, healthyProject())
		d.Slug = "demo"
		d.BillingCreds = &fakeBillingCreds{stripe: &adminv1.StripeBillingConfig{HasSecretKey: true}}
		d.Products = &fakeProducts{products: testProducts()}

		rep, err := d.Run(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if c := findCheck(t, rep, "project: Stripe webhook secret"); c.Status != StatusWarn {
			t.Fatalf("expected warn, got %+v", c)
		}
	})
}
