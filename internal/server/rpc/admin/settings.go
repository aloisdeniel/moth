package adminrpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	netmail "net/mail"
	"strings"
	"time"

	"connectrpc.com/connect"

	adminv1 "github.com/aloisdeniel/moth/gen/moth/admin/v1"
	"github.com/aloisdeniel/moth/internal/config"
	mailpkg "github.com/aloisdeniel/moth/internal/mail"
	"github.com/aloisdeniel/moth/internal/store"
	"github.com/aloisdeniel/moth/internal/version"
)

// SettingsHandler implements moth.admin.v1.InstanceSettingsService.
//
// The effective SMTP configuration is resolved with this precedence:
// database (set through this service) > config file / environment > none
// (console transport). Updates swap the shared dynamic mailer in place, so
// no restart is needed.
type SettingsHandler struct {
	store Store
	cfg   config.Config
	// dyn is the transport used by every email the server sends.
	dyn *mailpkg.Dynamic
	// fallback is what dyn falls back to when no SMTP is configured
	// anywhere (the console logger, or a recording mailer in tests).
	fallback mailpkg.Mailer
	now      func() time.Time
}

// NewSettingsHandler builds the instance settings service and points dyn
// at the effective SMTP transport.
func NewSettingsHandler(ctx context.Context, st Store, cfg config.Config, dyn *mailpkg.Dynamic, fallback mailpkg.Mailer) (*SettingsHandler, error) {
	h := &SettingsHandler{store: st, cfg: cfg, dyn: dyn, fallback: fallback, now: time.Now}
	smtp, source, err := h.effectiveSMTP(ctx)
	if err != nil {
		return nil, err
	}
	h.apply(smtp, source)
	return h, nil
}

// SMTPConfigured reports whether a real SMTP transport is currently
// effective (used to decide whether invite emails actually go out).
func (h *SettingsHandler) SMTPConfigured() bool {
	smtp, _, err := h.effectiveSMTP(context.Background())
	return err == nil && smtp.Enabled()
}

func (h *SettingsHandler) GetInstanceSettings(ctx context.Context, _ *connect.Request[adminv1.GetInstanceSettingsRequest]) (*connect.Response[adminv1.GetInstanceSettingsResponse], error) {
	smtp, source, err := h.effectiveSMTP(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&adminv1.GetInstanceSettingsResponse{
		BaseUrl:         strings.TrimSuffix(h.cfg.BaseURL, "/"),
		Version:         version.Version,
		Smtp:            smtpProto(smtp),
		SmtpSource:      source,
		SmtpHasPassword: smtp.Password != "",
	}), nil
}

func (h *SettingsHandler) UpdateSmtpSettings(ctx context.Context, req *connect.Request[adminv1.UpdateSmtpSettingsRequest]) (*connect.Response[adminv1.UpdateSmtpSettingsResponse], error) {
	msg := req.Msg.Smtp
	if msg == nil || strings.TrimSpace(msg.Host) == "" {
		// Clear the database override and fall back to config / console.
		if err := h.store.DeleteInstanceSetting(ctx, store.InstanceSettingSMTP); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	} else {
		smtp := config.SMTP{
			Host:     strings.TrimSpace(msg.Host),
			Port:     int(msg.Port),
			Username: strings.TrimSpace(msg.Username),
			Password: msg.Password,
			From:     strings.TrimSpace(msg.From),
		}
		if smtp.Port == 0 {
			smtp.Port = config.DefaultSMTPPort
		}
		if smtp.Port < 1 || smtp.Port > 65535 {
			return nil, connect.NewError(connect.CodeInvalidArgument,
				errors.New("port must be between 1 and 65535"))
		}
		if _, err := netmail.ParseAddress(smtp.From); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument,
				errors.New("a valid sender address (from) is required"))
		}
		if smtp.Password == "" {
			// Keep the previously stored password on edits that leave the
			// field blank.
			if prev, err := h.storedSMTP(ctx); err == nil {
				smtp.Password = prev.Password
			} else if !errors.Is(err, store.ErrNotFound) {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
		}
		raw, err := json.Marshal(smtp)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if err := h.store.SetInstanceSetting(ctx, store.InstanceSettingSMTP, string(raw), h.now()); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	smtp, source, err := h.effectiveSMTP(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	h.apply(smtp, source)
	return connect.NewResponse(&adminv1.UpdateSmtpSettingsResponse{
		Smtp:            smtpProto(smtp),
		SmtpSource:      source,
		SmtpHasPassword: smtp.Password != "",
	}), nil
}

func (h *SettingsHandler) SendTestEmail(ctx context.Context, req *connect.Request[adminv1.SendTestEmailRequest]) (*connect.Response[adminv1.SendTestEmailResponse], error) {
	to := strings.ToLower(strings.TrimSpace(req.Msg.To))
	if _, err := netmail.ParseAddress(to); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("invalid email address"))
	}
	if err := h.dyn.Send(ctx, mailpkg.Test(to)); err != nil {
		return nil, connect.NewError(connect.CodeUnavailable,
			fmt.Errorf("send test email: %w", err))
	}
	return connect.NewResponse(&adminv1.SendTestEmailResponse{}), nil
}

// storedSMTP reads the database SMTP override.
func (h *SettingsHandler) storedSMTP(ctx context.Context) (config.SMTP, error) {
	raw, err := h.store.GetInstanceSetting(ctx, store.InstanceSettingSMTP)
	if err != nil {
		return config.SMTP{}, err
	}
	var smtp config.SMTP
	if err := json.Unmarshal([]byte(raw), &smtp); err != nil {
		return config.SMTP{}, fmt.Errorf("parse stored smtp settings: %w", err)
	}
	return smtp, nil
}

// effectiveSMTP resolves the SMTP configuration and where it comes from.
func (h *SettingsHandler) effectiveSMTP(ctx context.Context) (config.SMTP, adminv1.SmtpSource, error) {
	smtp, err := h.storedSMTP(ctx)
	if err == nil && smtp.Enabled() {
		return smtp, adminv1.SmtpSource_SMTP_SOURCE_DATABASE, nil
	}
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return config.SMTP{}, adminv1.SmtpSource_SMTP_SOURCE_NONE, err
	}
	if h.cfg.SMTP.Enabled() {
		return h.cfg.SMTP, adminv1.SmtpSource_SMTP_SOURCE_CONFIG, nil
	}
	return config.SMTP{}, adminv1.SmtpSource_SMTP_SOURCE_NONE, nil
}

// apply points the shared dynamic mailer at the transport smtp describes.
func (h *SettingsHandler) apply(smtp config.SMTP, source adminv1.SmtpSource) {
	if source == adminv1.SmtpSource_SMTP_SOURCE_NONE || !smtp.Enabled() {
		h.dyn.Set(h.fallback)
		return
	}
	h.dyn.Set(mailpkg.SMTP{
		Host:     smtp.Host,
		Port:     smtp.Port,
		Username: smtp.Username,
		Password: smtp.Password,
		From:     smtp.From,
	})
}

// smtpProto converts to the wire message with the password blanked
// (write-only field).
func smtpProto(s config.SMTP) *adminv1.SmtpSettings {
	return &adminv1.SmtpSettings{
		Host:     s.Host,
		Port:     int32(s.Port),
		Username: s.Username,
		From:     s.From,
	}
}
