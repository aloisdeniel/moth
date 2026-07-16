package cli

import (
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
)

// The --json output contract: byte-stable JSON (protojson randomizes its
// whitespace on purpose; scripts and this golden test rely on the
// re-indented form staying fixed).
func TestMarshalJSONGolden(t *testing.T) {
	autoLink := true
	project := &adminv1.Project{
		Id:             "0198f2c4-0000-7000-8000-000000000001",
		Name:           "Demo App",
		Slug:           "demo-app",
		PublishableKey: "moth_pk_test",
		CreateTime:     timestamppb.New(time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)),
		UserCount:      2,
		Settings: &adminv1.ProjectSettings{
			PasswordMinLength:     8,
			AllowPublicSignup:     true,
			AccessTokenTtlSeconds: 900,
			RefreshTokenTtlDays:   30,
			Google:                &adminv1.GoogleProviderConfig{Enabled: true, WebClientId: "web.apps.example"},
			AutoLinkVerifiedEmail: &autoLink,
		},
	}

	const want = `{
  "id": "0198f2c4-0000-7000-8000-000000000001",
  "name": "Demo App",
  "slug": "demo-app",
  "publishableKey": "moth_pk_test",
  "createTime": "2026-07-01T12:00:00Z",
  "settings": {
    "passwordMinLength": 8,
    "allowPublicSignup": true,
    "accessTokenTtlSeconds": 900,
    "refreshTokenTtlDays": 30,
    "google": {
      "enabled": true,
      "webClientId": "web.apps.example"
    },
    "autoLinkVerifiedEmail": true
  },
  "userCount": "2"
}
`
	for range 20 { // protojson whitespace randomization must never leak through
		got, err := MarshalJSON(project)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != want {
			t.Fatalf("golden mismatch:\n got: %s\nwant: %s", got, want)
		}
	}
}

func TestMarshalJSONGoldenPAT(t *testing.T) {
	resp := &adminv1.ListPersonalAccessTokensResponse{
		Tokens: []*adminv1.PersonalAccessToken{{
			Id:         "0198f2c4-0000-7000-8000-000000000002",
			Name:       "ci",
			CreateTime: timestamppb.New(time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)),
			ExpireTime: timestamppb.New(time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)),
		}},
	}
	const want = `{
  "tokens": [
    {
      "id": "0198f2c4-0000-7000-8000-000000000002",
      "name": "ci",
      "createTime": "2026-07-01T12:00:00Z",
      "expireTime": "2026-07-31T12:00:00Z"
    }
  ]
}
`
	got, err := MarshalJSON(resp)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("golden mismatch:\n got: %s\nwant: %s", got, want)
	}
}

// TestMarshalJSONGoldenCommandShapes pins the --json output of the
// remaining command groups (stats get, instance get, user get, project
// keys show, project keys regenerate-secret): a field rename in any of
// these breaks scripts and the exported skill's documented contracts.
func TestMarshalJSONGoldenCommandShapes(t *testing.T) {
	ts := timestamppb.New(time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC))
	cases := []struct {
		name string
		msg  proto.Message
		want string
	}{
		{
			name: "stats get",
			msg: &adminv1.GetStatsResponse{
				Tiles: &adminv1.StatTiles{
					TotalUsers: 42, NewUsers_7D: 5, NewUsersPrevious_7D: 3,
					LatestDau: 7, LatestDauDate: "2026-06-30",
					Logins_7D: 120, LoginFailures_7D: 4, LoginSuccessRate_7D: 0.75,
				},
				Series:    []*adminv1.DailyStat{{Date: "2026-06-30", Signups: 1, Logins: 20, Dau: 7, Failures: 1}},
				Providers: &adminv1.ProviderBreakdown{Password: 100, Google: 15, Apple: 5},
				Platforms: &adminv1.PlatformBreakdown{Ios: 60, Android: 50, Web: 10},
			},
			want: `{
  "tiles": {
    "totalUsers": "42",
    "newUsers7d": "5",
    "newUsersPrevious7d": "3",
    "latestDau": "7",
    "latestDauDate": "2026-06-30",
    "logins7d": "120",
    "loginFailures7d": "4",
    "loginSuccessRate7d": 0.75
  },
  "series": [
    {
      "date": "2026-06-30",
      "signups": "1",
      "logins": "20",
      "dau": "7",
      "failures": "1"
    }
  ],
  "providers": {
    "password": "100",
    "google": "15",
    "apple": "5"
  },
  "platforms": {
    "ios": "60",
    "android": "50",
    "web": "10"
  }
}
`,
		},
		{
			name: "instance get",
			msg: &adminv1.GetInstanceSettingsResponse{
				BaseUrl: "https://auth.example.com", Version: "dev",
				Smtp:       &adminv1.SmtpSettings{Host: "smtp.example.com", Port: 587, Username: "mailer", From: "auth@example.com"},
				SmtpSource: adminv1.SmtpSource_SMTP_SOURCE_DATABASE, SmtpHasPassword: true,
			},
			want: `{
  "baseUrl": "https://auth.example.com",
  "version": "dev",
  "smtp": {
    "host": "smtp.example.com",
    "port": 587,
    "username": "mailer",
    "from": "auth@example.com"
  },
  "smtpSource": "SMTP_SOURCE_DATABASE",
  "smtpHasPassword": true
}
`,
		},
		{
			name: "user get",
			msg: &adminv1.GetUserResponse{
				User: &adminv1.User{
					Id: "0198f2c4-0000-7000-8000-000000000003", Email: "user@example.com",
					EmailVerified: true, DisplayName: "User", CreateTime: ts,
					Providers: []string{"password", "google"}, CustomClaims: `{"role":"admin"}`,
				},
				Sessions:   []*adminv1.UserSession{{Id: "s1", CreateTime: ts}},
				Identities: []*adminv1.Identity{{Provider: "password", CreateTime: ts}},
			},
			want: `{
  "user": {
    "id": "0198f2c4-0000-7000-8000-000000000003",
    "email": "user@example.com",
    "emailVerified": true,
    "displayName": "User",
    "createTime": "2026-07-01T12:00:00Z",
    "providers": [
      "password",
      "google"
    ],
    "customClaims": "{\"role\":\"admin\"}"
  },
  "sessions": [
    {
      "id": "s1",
      "createTime": "2026-07-01T12:00:00Z"
    }
  ],
  "identities": [
    {
      "provider": "password",
      "createTime": "2026-07-01T12:00:00Z"
    }
  ]
}
`,
		},
		{
			name: "project keys show",
			msg: &adminv1.GetSigningKeyResponse{
				Key:     &adminv1.SigningKey{Kid: "kid123", Algorithm: "ES256", PublicKeyPem: "PEM", CreateTime: ts},
				JwksUrl: "https://auth.example.com/p/demo-app/.well-known/jwks.json",
				Issuer:  "https://auth.example.com/p/demo-app", Audience: "demo-app",
			},
			want: `{
  "key": {
    "kid": "kid123",
    "algorithm": "ES256",
    "publicKeyPem": "PEM",
    "createTime": "2026-07-01T12:00:00Z"
  },
  "jwksUrl": "https://auth.example.com/p/demo-app/.well-known/jwks.json",
  "issuer": "https://auth.example.com/p/demo-app",
  "audience": "demo-app"
}
`,
		},
		{
			name: "project keys regenerate-secret",
			msg: &adminv1.RegenerateSecretKeyResponse{
				Project: &adminv1.Project{
					Id: "0198f2c4-0000-7000-8000-000000000001", Name: "Demo App",
					Slug: "demo-app", PublishableKey: "moth_pk_test", CreateTime: ts,
				},
				SecretKey: "sk_test_secret",
			},
			want: `{
  "project": {
    "id": "0198f2c4-0000-7000-8000-000000000001",
    "name": "Demo App",
    "slug": "demo-app",
    "publishableKey": "moth_pk_test",
    "createTime": "2026-07-01T12:00:00Z"
  },
  "secretKey": "sk_test_secret"
}
`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := MarshalJSON(tc.msg)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Fatalf("golden mismatch:\n got: %s\nwant: %s", got, tc.want)
			}
		})
	}
}
