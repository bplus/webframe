// templui component icon - version: v1.3.0 installed by templui v1.3.0
// 📚 Documentation: https://templui.io/docs/components/icon
package icon

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/a-h/templ"
)

// iconContents caches the fully generated SVG strings for icons that have been used,
// keyed by a composite key of name and props to handle different stylings.
// Guarded by iconMutex. The cache grows unbounded — one entry per unique
// name+props combination for the life of the process — which is fine for the
// intended use (a finite set of icons/stylings per app), but means callers
// should not feed it unbounded user-derived props.
var (
	iconContents = make(map[string]string)
	iconMutex    sync.RWMutex
)

// Props defines the properties that can be set for an icon.
//
// The zero value is valid and renders the Lucide defaults: Size <= 0 becomes
// 24, empty Fill becomes "none", empty Stroke falls back to Color and then to
// "currentColor", and empty StrokeWidth becomes "2" (all applied in
// generateSVG at render time). Values are interpolated into SVG attributes
// without sanitization, so props must come from trusted (developer) code, not
// end-user input.
type Props struct {
	Size        int
	Color       string
	Fill        string
	Stroke      string
	StrokeWidth string // Stroke Width of Icon, Usage: "2.5"
	Class       string
}

// Icon returns a function that generates a templ.Component for the specified icon name.
// This two-step shape exists so the generated icon_defs.go can predefine
// ~1,600 icons as `var X = Icon("x")` while apps still choose Props per use.
// The component writes a complete <svg> element, memoized in the package
// cache so repeated renders of the same name+props are a single map read.
//
// An unknown name is not detected here — the error surfaces at render time,
// wrapped with the icon name and props. Only the first Props value is used;
// extras are silently ignored.
//
// @thread-safe - cache guarded by iconMutex (sync.RWMutex); two goroutines
// racing on an uncached icon may both generate it, but the writes are
// identical so the race is benign
// @side-effects inserts into the package-level iconContents cache
// @complexity O(1) amortized per render after first generation
func Icon(name string) func(...Props) templ.Component {
	return func(props ...Props) templ.Component {
		var p Props
		if len(props) > 0 {
			p = props[0]
		}

		// Create a unique key for the cache based on icon name and all relevant props.
		// This ensures different stylings of the same icon are cached separately.
		cacheKey := fmt.Sprintf("%s|s:%d|c:%s|f:%s|sk:%s|sw:%s|cl:%s",
			name, p.Size, p.Color, p.Fill, p.Stroke, p.StrokeWidth, p.Class)

		return templ.ComponentFunc(func(ctx context.Context, w io.Writer) (err error) {
			iconMutex.RLock()
			svg, cached := iconContents[cacheKey]
			iconMutex.RUnlock()

			if cached {
				_, err = w.Write([]byte(svg))
				return err
			}

			// Not cached, generate it
			// The actual generation now happens once and is cached.
			generatedSvg, err := generateSVG(name, p) // p (Props) is passed to generateSVG
			if err != nil {
				// Provide more context in the error message
				return fmt.Errorf("failed to generate svg for icon '%s' with props %+v: %w", name, p, err)
			}

			iconMutex.Lock()
			iconContents[cacheKey] = generatedSvg
			iconMutex.Unlock()

			_, err = w.Write([]byte(generatedSvg))
			return err
		})
	}
}

// generateSVG creates an SVG string for the specified icon with the given properties.
// This function is called when an icon-prop combination is not yet in the cache.
// Returns an error only when the name is missing from internalSvgData; all
// prop defaulting (size, fill, stroke, stroke-width) happens here.
//
// @pure - reads only the immutable generated internalSvgData map
func generateSVG(name string, props Props) (string, error) {
	// Get the raw, inner SVG content for the icon name from our internal data map.
	content, err := getIconContent(name) // This now reads from internalSvgData
	if err != nil {
		return "", err // Error from getIconContent already includes icon name
	}

	size := props.Size
	if size <= 0 {
		size = 24 // Default size
	}

	fill := props.Fill
	if fill == "" {
		fill = "none" // Default fill
	}

	stroke := props.Stroke
	if stroke == "" {
		stroke = props.Color // Fallback to Color if Stroke is not set
	}
	if stroke == "" {
		stroke = "currentColor" // Default stroke color
	}

	strokeWidth := props.StrokeWidth
	if strokeWidth == "" {
		strokeWidth = "2" // Default stroke width
	}

	// Construct the final SVG string.
	// The data-lucide attribute helps identify these as Lucide icons if needed.
	return fmt.Sprintf("<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"%d\" height=\"%d\" viewBox=\"0 0 24 24\" fill=\"%s\" stroke=\"%s\" stroke-width=\"%s\" stroke-linecap=\"round\" stroke-linejoin=\"round\" class=\"%s\" data-lucide=\"icon\">%s</svg>",
		size, size, fill, stroke, strokeWidth, props.Class, content), nil
}

// getIconContent retrieves the raw inner SVG content for a given icon name.
// It reads from the pre-generated internalSvgData map from icon_data.go.
// A miss (name not in the Lucide set, or a typo) is the only error path.
//
// @pure
func getIconContent(name string) (string, error) {
	content, exists := internalSvgData[name]
	if !exists {
		return "", fmt.Errorf("icon '%s' not found in internalSvgData map", name)
	}
	return content, nil
}
