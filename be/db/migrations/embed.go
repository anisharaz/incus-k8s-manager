// Package migrations embeds the SQL migration files into the compiled
// binary so the app can apply them itself on startup (see runMigrations in
// cmd/server/main.go) without depending on the standalone migrate CLI.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
