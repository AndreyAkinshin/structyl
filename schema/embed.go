// Package schema provides embedded JSON schemas for structyl configuration files.
package schema

import "embed"

// FS contains the embedded schema files.
//
//go:embed *.schema.json
var FS embed.FS
