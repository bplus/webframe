// Package graphwidget renders an interactive 3D node-and-edge graph that
// loads its data as JSON from a URL. Use it to explore relationships between
// records; it ships its own script, which the page loads through Script.
//
// Needs JavaScript: the graph is drawn by script (with WebGL), so without it
// the panel shows its loading spinner forever.
package graphwidget
