package migrations

import "embed"

// FS contains the exact SQL migrations used by the one-off migration process.
//
//go:embed *.sql
var FS embed.FS
