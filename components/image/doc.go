// Package image renders the <img> every app picture goes through: the source
// sanitized, alt text always present, optional width and height, lazy loading
// by default. App code cannot write a raw <img> (the wflint linter forbids
// it), so it uses this.
package image
