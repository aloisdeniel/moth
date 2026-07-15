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

// layoutData feeds both the plain-text and HTML layout templates.
type layoutData struct {
	Project     string
	Subject     string
	Paragraphs  []string
	ButtonLabel string
	ButtonURL   string
}

// Verification is the "confirm your email address" email.
func Verification(project, to, link string) Message {
	return render(to, layoutData{
		Project: project,
		Subject: fmt.Sprintf("Verify your email for %s", project),
		Paragraphs: []string{
			fmt.Sprintf("Confirm this email address to finish setting up your %s account.", project),
			"If you did not create this account, you can ignore this email.",
		},
		ButtonLabel: "Verify email",
		ButtonURL:   link,
	})
}

// PasswordReset is the "reset your password" email.
func PasswordReset(project, to, link string) Message {
	return render(to, layoutData{
		Project: project,
		Subject: fmt.Sprintf("Reset your %s password", project),
		Paragraphs: []string{
			fmt.Sprintf("A password reset was requested for your %s account.", project),
			"If you did not request this, you can ignore this email — your password is unchanged.",
		},
		ButtonLabel: "Reset password",
		ButtonURL:   link,
	})
}

// EmailChangeConfirm goes to the NEW address to prove ownership before the
// account email switches.
func EmailChangeConfirm(project, to, link string) Message {
	return render(to, layoutData{
		Project: project,
		Subject: fmt.Sprintf("Confirm your new email for %s", project),
		Paragraphs: []string{
			fmt.Sprintf("Confirm that you want to use this address for your %s account.", project),
			"If you did not request this change, you can ignore this email.",
		},
		ButtonLabel: "Confirm new email",
		ButtonURL:   link,
	})
}

// EmailChangedNotice goes to the OLD address after the switch, carrying a
// revert link.
func EmailChangedNotice(project, to, newEmail, revertLink string) Message {
	return render(to, layoutData{
		Project: project,
		Subject: fmt.Sprintf("Your %s email address was changed", project),
		Paragraphs: []string{
			fmt.Sprintf("The email address on your %s account was changed to %s.", project, newEmail),
			"If this was you, no action is needed.",
			"If you did not make this change, you can restore this address within 72 hours:",
		},
		ButtonLabel: "Restore this email",
		ButtonURL:   revertLink,
	})
}

// AccountExists is sent instead of an error when an enumeration-safe
// project sees a signup with an already-registered email.
func AccountExists(project, to string) Message {
	return render(to, layoutData{
		Project: project,
		Subject: fmt.Sprintf("You already have a %s account", project),
		Paragraphs: []string{
			fmt.Sprintf("Someone (probably you) tried to sign up for %s with this email address, but an account already exists.", project),
			"You can simply sign in — and use the \"forgot password\" option if you no longer remember your password.",
			"If this was not you, no action is needed; your account is unchanged.",
		},
	})
}

func render(to string, data layoutData) Message {
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
