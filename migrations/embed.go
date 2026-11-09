// Package migrations embeds the SQL files so cmd/migrate needs no path handling.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
