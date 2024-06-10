package migrations

import "embed"

//go:embed *.sql
var SQLs embed.FS
