// Package migrations embeds Oryn's versioned SQL migrations.
package migrations

import "embed"

// FS contains all migration files.
//
//go:embed *.sql
var FS embed.FS
