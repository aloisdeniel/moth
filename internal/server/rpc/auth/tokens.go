package authrpc

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"time"

	authv1 "github.com/aloisdeniel/moth/gen/moth/auth/v1"
	"github.com/aloisdeniel/moth/internal/jwt"
	"github.com/aloisdeniel/moth/internal/keys"
	"github.com/aloisdeniel/moth/internal/store"
	"github.com/aloisdeniel/moth/internal/token"
)

// Issuer returns the iss claim for a project: the instance base URL plus
// the project slug.
func Issuer(baseURL, slug string) string {
	return baseURL + "/p/" + slug
}

// signingKey returns the newest active key of the project with its
// decrypted private part.
func (h *Handler) signingKey(ctx context.Context, projectID string) (store.ProjectKey, *ecdsa.PrivateKey, error) {
	active, err := h.store.ListActiveProjectKeys(ctx, projectID)
	if err != nil {
		return store.ProjectKey{}, nil, fmt.Errorf("list project keys: %w", err)
	}
	if len(active) == 0 {
		return store.ProjectKey{}, nil, fmt.Errorf("project %s has no active signing key", projectID)
	}
	newest := active[len(active)-1] // ListActiveProjectKeys orders by created_at
	priv, err := keys.DecryptPrivateKey(h.master, newest.PrivateKeyEnc)
	if err != nil {
		return store.ProjectKey{}, nil, fmt.Errorf("decrypt signing key: %w", err)
	}
	return newest, priv, nil
}

// publicKeyLookup resolves a kid to one of the project's active public
// keys, for access-token verification.
func (h *Handler) publicKeyLookup(ctx context.Context, projectID string) func(kid string) (*ecdsa.PublicKey, error) {
	return func(kid string) (*ecdsa.PublicKey, error) {
		active, err := h.store.ListActiveProjectKeys(ctx, projectID)
		if err != nil {
			return nil, err
		}
		for _, k := range active {
			if k.Kid == kid {
				return keys.ParsePublicKeyPEM(k.PublicKeyPEM)
			}
		}
		return nil, fmt.Errorf("unknown kid %q", kid)
	}
}

// mintAccessToken signs a fresh JWT for the user with the project's key.
func (h *Handler) mintAccessToken(ctx context.Context, project store.Project, user store.User) (string, int64, error) {
	key, priv, err := h.signingKey(ctx, project.ID)
	if err != nil {
		return "", 0, err
	}
	now := h.now()
	ttl := time.Duration(project.Settings.AccessTokenTTLSeconds) * time.Second
	claims := jwt.Claims{
		Issuer:        Issuer(h.baseURL, project.Slug),
		Subject:       user.ID,
		Audience:      project.Slug,
		IssuedAt:      now.Unix(),
		ExpiresAt:     now.Add(ttl).Unix(),
		Email:         user.Email,
		EmailVerified: user.Verified(),
		Custom:        customClaims(user),
	}
	signed, err := jwt.Sign(priv, key.Kid, claims)
	if err != nil {
		return "", 0, err
	}
	return signed, int64(ttl.Seconds()), nil
}

// issueSession starts a new refresh-token family for the user and mints
// the first access token — one sign-in on one device.
func (h *Handler) issueSession(ctx context.Context, project store.Project, user store.User, deviceInfo string) (*authv1.TokenPair, error) {
	refresh := token.Random(32) // 256-bit opaque token
	now := h.now()
	rt := store.RefreshToken{
		ID:         NewID(),
		ProjectID:  project.ID,
		UserID:     user.ID,
		TokenHash:  hashToken(refresh),
		FamilyID:   NewID(),
		DeviceInfo: deviceInfo,
		ExpiresAt:  now.Add(h.refreshTTL(project)),
		CreatedAt:  now,
	}
	if err := h.store.CreateRefreshToken(ctx, rt); err != nil {
		return nil, err
	}
	access, expiresIn, err := h.mintAccessToken(ctx, project, user)
	if err != nil {
		return nil, err
	}
	return &authv1.TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    expiresIn,
	}, nil
}

func (h *Handler) refreshTTL(project store.Project) time.Duration {
	return time.Duration(project.Settings.RefreshTokenTTLDays) * 24 * time.Hour
}
