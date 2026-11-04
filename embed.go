// Package webframe is the module root. It only embeds the static/ directory,
// since //go:embed cannot reach a parent directory. Apps do not import it:
// they serve the files with package assets and build pages from the packages
// under components.
package webframe

import "embed"

// Static holds the library's CSS, JS, fonts and icons.
//
//go:embed static
var Static embed.FS
