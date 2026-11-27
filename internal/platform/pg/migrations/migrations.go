// Package migrations embeds the versioned goose migrations (NFR-MNT-02).
// Files are applied in the order of their numeric prefix; never edit a
// migration that is already in main, add a new one instead.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
