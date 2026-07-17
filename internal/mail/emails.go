package mail

import (
	"bytes"
	"embed"
	"fmt"
	htmltemplate "html/template"
	texttemplate "text/template"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

var (
	textLayout = texttemplate.Must(texttemplate.ParseFS(templateFS, "templates/layout.txt.tmpl"))
	htmlLayout = htmltemplate.Must(htmltemplate.ParseFS(templateFS, "templates/layout.html.tmpl"))
)

// Brand is the sender identity an email renders with: the project name
// plus the optional design-system accents (plan/06). The zero-value accents
// fall back to moth's neutral defaults, so a Brand{Name: ...} alone always
// produces a decent email.
type Brand struct {
	// Name of the project (or "moth" for instance-level mail).
	Name string
	// LogoURL is the absolute URL of the project's light-scheme logo;
	// empty renders no logo. Email clients overwhelmingly render on white,
	// so the light variant is the right one.
	LogoURL string
	// Accent is the button background (#RRGGBB); theme primary color.
	Accent string
	// OnAccent is the button text color (#RRGGBB); theme onPrimary.
	OnAccent string
}

// Neutral fallback accents for unbranded email (instance-level mail, or a
// project that somehow has no theme).
const (
	defaultAccent   = "#1a1a1a"
	defaultOnAccent = "#ffffff"
)

// layoutData feeds both the plain-text and HTML layout templates.
type layoutData struct {
	Project     string
	LogoURL     string
	Accent      string
	OnAccent    string
	Subject     string
	Paragraphs  []string
	ButtonLabel string
	ButtonURL   string
}

// Verification is the "confirm your email address" email.
func Verification(brand Brand, to, link string) Message {
	return render(to, brand, layoutData{
		Subject: fmt.Sprintf("Verify your email for %s", brand.Name),
		Paragraphs: []string{
			fmt.Sprintf("Confirm this email address to finish setting up your %s account.", brand.Name),
			"If you did not create this account, you can ignore this email.",
		},
		ButtonLabel: "Verify email",
		ButtonURL:   link,
	})
}

// PasswordReset is the "reset your password" email.
func PasswordReset(brand Brand, to, link string) Message {
	return render(to, brand, layoutData{
		Subject: fmt.Sprintf("Reset your %s password", brand.Name),
		Paragraphs: []string{
			fmt.Sprintf("A password reset was requested for your %s account.", brand.Name),
			"If you did not request this, you can ignore this email — your password is unchanged.",
		},
		ButtonLabel: "Reset password",
		ButtonURL:   link,
	})
}

// EmailChangeConfirm goes to the NEW address to prove ownership before the
// account email switches.
func EmailChangeConfirm(brand Brand, to, link string) Message {
	return render(to, brand, layoutData{
		Subject: fmt.Sprintf("Confirm your new email for %s", brand.Name),
		Paragraphs: []string{
			fmt.Sprintf("Confirm that you want to use this address for your %s account.", brand.Name),
			"If you did not request this change, you can ignore this email.",
		},
		ButtonLabel: "Confirm new email",
		ButtonURL:   link,
	})
}

// EmailChangedNotice goes to the OLD address after the switch, carrying a
// revert link.
func EmailChangedNotice(brand Brand, to, newEmail, revertLink string) Message {
	return render(to, brand, layoutData{
		Subject: fmt.Sprintf("Your %s email address was changed", brand.Name),
		Paragraphs: []string{
			fmt.Sprintf("The email address on your %s account was changed to %s.", brand.Name, newEmail),
			"If this was you, no action is needed.",
			"If you did not make this change, you can restore this address within 72 hours:",
		},
		ButtonLabel: "Restore this email",
		ButtonURL:   revertLink,
	})
}

// AccountExists is sent instead of an error when an enumeration-safe
// project sees a signup with an already-registered email.
func AccountExists(brand Brand, to string) Message {
	return render(to, brand, layoutData{
		Subject: fmt.Sprintf("You already have a %s account", brand.Name),
		Paragraphs: []string{
			fmt.Sprintf("Someone (probably you) tried to sign up for %s with this email address, but an account already exists.", brand.Name),
			"You can simply sign in — and use the \"forgot password\" option if you no longer remember your password.",
			"If this was not you, no action is needed; your account is unchanged.",
		},
	})
}

// UserInvite is the "set your password" email for an account created by an
// operator in the admin console.
func UserInvite(brand Brand, to, link string) Message {
	return render(to, brand, layoutData{
		Subject: fmt.Sprintf("You've been invited to %s", brand.Name),
		Paragraphs: []string{
			fmt.Sprintf("An account was created for you on %s.", brand.Name),
			"Choose a password to start using it:",
		},
		ButtonLabel: "Set your password",
		ButtonURL:   link,
	})
}

// AdminInvite is the operator-invitation email for the moth admin console.
func AdminInvite(to, link string) Message {
	return render(to, Brand{Name: "moth"}, layoutData{
		Subject: "You've been invited to administer a moth instance",
		Paragraphs: []string{
			"You've been invited to become an operator of a moth instance.",
			"Open the link below to choose a password and activate your admin account.",
			"If you were not expecting this invitation, you can ignore this email.",
		},
		ButtonLabel: "Activate admin account",
		ButtonURL:   link,
	})
}

// Test is the probe email sent by the admin console's "send test email"
// button.
func Test(to string) Message {
	return render(to, Brand{Name: "moth"}, layoutData{
		Subject: "moth test email",
		Paragraphs: []string{
			"This is a test email from your moth instance.",
			"If you can read this, outgoing email is configured correctly.",
		},
	})
}

// Content is a pre-localized transactional email: the subject, body
// paragraphs and an optional action button, already resolved from the i18n
// catalog for the recipient's negotiated locale by the caller. It lets the
// auth handlers ship localized copy through the same branded layout the
// English helpers above use.
type Content struct {
	Subject     string
	Paragraphs  []string
	ButtonLabel string
	ButtonURL   string
}

// RenderContent builds a branded email from pre-resolved localized content.
func RenderContent(brand Brand, to string, c Content) Message {
	return render(to, brand, layoutData{
		Subject:     c.Subject,
		Paragraphs:  c.Paragraphs,
		ButtonLabel: c.ButtonLabel,
		ButtonURL:   c.ButtonURL,
	})
}

func render(to string, brand Brand, data layoutData) Message {
	data.Project = brand.Name
	data.LogoURL = brand.LogoURL
	data.Accent = brand.Accent
	data.OnAccent = brand.OnAccent
	if data.Accent == "" {
		data.Accent = defaultAccent
	}
	if data.OnAccent == "" {
		data.OnAccent = defaultOnAccent
	}
	var text, html bytes.Buffer
	// The layouts are static and the data is server-built, so rendering
	// can only fail on a programming error.
	if err := textLayout.Execute(&text, data); err != nil {
		panic(fmt.Sprintf("render text email: %v", err))
	}
	if err := htmlLayout.Execute(&html, data); err != nil {
		panic(fmt.Sprintf("render html email: %v", err))
	}
	return Message{To: to, Subject: data.Subject, Text: text.String(), HTML: html.String()}
}
