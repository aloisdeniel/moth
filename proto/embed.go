// Package proto embeds the .proto sources so the server can offer them for
// download from the admin setup-instructions page (developers generate
// their own moth.server.v1 clients from these).
package proto

import "embed"

// FS holds every .proto file under proto/moth.
//
//go:embed moth
var FS embed.FS
