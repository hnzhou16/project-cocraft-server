package mailer

import "embed"

const (
	FromName             = "CoCraft"
	maxRetires           = 3
	isSandbox            = true
	UserActivateTemplate = "user_invitation.tmpl"
)

// FS embed files in 'templates' folder
//
//go:embed "templates"
var FS embed.FS

type Client interface {
	Send(templateFile, username, email string, data any) (int, error)
}
