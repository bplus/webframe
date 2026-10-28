// Package webframe is the module root. It exists to embed static/, since
// //go:embed cannot reach a parent directory.
package webframe

import "embed"

// Static holds the library's CSS, JS, fonts and icons.
//
//go:embed static
var Static embed.FS
