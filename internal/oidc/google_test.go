package oidc

import (
	"net/http"
	"testing"
)

func TestGoogleExchangeCode(t *testing.T) {
	endpoint := newTokenEndpointDouble(t, http.StatusOK,
		`{"access_token":"at","token_type":"Bearer","expires_in":3599,"id_token":"google-idt"}`)
	client := NewGoogleClient(endpoint.srv.URL+"/token", "client-1", "secret-1", endpoint.srv.Client())

	tokens, err := client.ExchangeCode(t.Context(), "code-1", "https://moth.example.com/oauth/google/callback")
	if err != nil {
		t.Fatalf("ExchangeCode() error = %v", err)
	}
	if tokens.IDToken != "google-idt" {
		t.Errorf("IDToken = %q, want google-idt", tokens.IDToken)
	}
	form := endpoint.forms["/token"]
	if form == nil {
		t.Fatal("no POST to /token")
	}
	want := map[string]string{
		"grant_type":    "authorization_code",
		"code":          "code-1",
		"client_id":     "client-1",
		"client_secret": "secret-1",
		"redirect_uri":  "https://moth.example.com/oauth/google/callback",
	}
	for k, v := range want {
		if got := form.Get(k); got != v {
			t.Errorf("form[%s] = %q, want %q", k, got, v)
		}
	}
}
