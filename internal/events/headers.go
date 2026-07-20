package events

import (
	"context"
	"net/http"
	"strings"
)

// Request-metadata headers attached by the SDK client interceptor
// (milestone 05) on every call.
const (
	PlatformHeader   = "x-moth-platform"
	SDKVersionHeader = "x-moth-sdk-version"
)

// Platforms recognized in x-moth-platform. Anything else non-empty (a
// future OS, garbage) is bucketed as "other"; a missing header stays "".
const (
	PlatformIOS     = "ios"
	PlatformAndroid = "android"
	PlatformWeb     = "web"
	PlatformMacOS   = "macos"
	PlatformWindows = "windows"
	PlatformLinux   = "linux"
	PlatformOther   = "other"
)

// maxSDKVersionLen caps x-moth-sdk-version; longer or malformed values are
// discarded rather than stored.
const maxSDKVersionLen = 32

// ClientInfo is the ambient client context extracted from request headers.
type ClientInfo struct {
	Platform   string // one of the Platform* constants, or ""
	SDKVersion string // sanitized version string, or ""
}

type clientInfoKey struct{}

// WithClientInfo stashes info in ctx for the event constructors to pick
// up. The auth interceptor calls this once per request.
func WithClientInfo(ctx context.Context, info ClientInfo) context.Context {
	return context.WithValue(ctx, clientInfoKey{}, info)
}

// ClientInfoFromContext returns the info stored by WithClientInfo, or the
// zero value when none was stored.
func ClientInfoFromContext(ctx context.Context) ClientInfo {
	info, _ := ctx.Value(clientInfoKey{}).(ClientInfo)
	return info
}

// ClientInfoFromHeader parses and validates the SDK metadata headers.
// connect exposes request metadata as http.Header on both gRPC and
// gRPC-web transports.
func ClientInfoFromHeader(h http.Header) ClientInfo {
	return ClientInfo{
		Platform:   ParsePlatform(h.Get(PlatformHeader)),
		SDKVersion: ParseSDKVersion(h.Get(SDKVersionHeader)),
	}
}

// ParsePlatform validates a raw x-moth-platform value against the known
// enum (case-insensitively). Unknown non-empty values become "other" so a
// hostile client cannot inject arbitrary strings into analytics; an absent
// header stays "".
func ParsePlatform(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "":
		return ""
	case PlatformIOS:
		return PlatformIOS
	case PlatformAndroid:
		return PlatformAndroid
	case PlatformWeb:
		return PlatformWeb
	case PlatformMacOS:
		return PlatformMacOS
	case PlatformWindows:
		return PlatformWindows
	case PlatformLinux:
		return PlatformLinux
	default:
		return PlatformOther
	}
}

// ParseSDKVersion sanitizes a raw x-moth-sdk-version value: trimmed,
// at most maxSDKVersionLen bytes, and restricted to the characters found
// in semver-ish strings ([0-9A-Za-z .+_-]). Anything else returns "" —
// a version header is best-effort context, never worth storing garbage.
func ParseSDKVersion(raw string) string {
	v := strings.TrimSpace(raw)
	if v == "" || len(v) > maxSDKVersionLen {
		return ""
	}
	for _, r := range v {
		switch {
		case r >= '0' && r <= '9',
			r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r == '.', r == '+', r == '-', r == '_', r == ' ':
		default:
			return ""
		}
	}
	return v
}
