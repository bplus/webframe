// templui component icon - version: v1.3.0 installed by templui v1.3.0
// 📚 Documentation: https://templui.io/docs/components/icon
package icon

import (
	"context"
	"fmt"
	"html"
	"io"
	"sync"
	"sync/atomic"

	"github.com/a-h/templ"
)

// iconCacheCap bounds the number of entries iconContents may hold. When a
// write would exceed it, the whole map is cleared first: regeneration from
// internalSvgData is cheap, so a full flush is simpler than an LRU and keeps
// memory bounded even if callers feed it many distinct Class values.
const iconCacheCap = 4096

// iconContents caches the fully generated SVG strings for icons that have been used,
// keyed by a composite key of name and props to handle different stylings.
// Guarded by iconMutex. Bounded to iconCacheCap entries: once full, the map is
// cleared and repopulated on demand, so a long-running server with many
// distinct name+props combinations (e.g. per-row Class values) does not grow
// this cache without limit.
var (
	iconContents = make(map[string]string)
	iconMutex    sync.RWMutex
)

// iconGenerations counts calls to generateSVG (i.e. cache misses). Exported
// for tests to verify a repeated render is served from cache rather than
// regenerated.
var iconGenerations atomic.Int64

// Props defines the properties that can be set for an icon.
//
// The zero value is valid and renders the Lucide defaults: Size <= 0 becomes
// 24, empty Fill becomes "none", empty Stroke falls back to Color and then to
// "currentColor", and empty StrokeWidth becomes "2" (all applied in
// generateSVG at render time). String values are HTML-escaped into their SVG
// attributes, so a quote cannot break out of one; they are not otherwise
// validated (a bogus color is passed through as text).
type Props struct {
	Size        int
	Color       string
	Fill        string
	Stroke      string
	StrokeWidth string // Stroke Width of Icon, Usage: "2.5"
	// Class is part of the cache key (see iconContents), so distinct values
	// each occupy a cache slot; the cache is bounded to iconCacheCap and
	// clears itself when full rather than growing without limit.
	//
	// Unlike most component Props.Class fields, this is written (escaped)
	// into the SVG's class attribute — it is not passed through TwMerge, so
	// conflicting utilities are not resolved; callers cannot rely on a later
	// class winning over the defaults.
	Class string
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
		// The values are quoted (%q) so the key is unambiguous: a value
		// containing "|f:" cannot spell another Props' key. The key holds
		// the raw values; generateSVG escapes deterministically, so equal
		// keys still mean equal markup.
		cacheKey := fmt.Sprintf("%q|s:%d|c:%q|f:%q|sk:%q|sw:%q|cl:%q",
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
			iconGenerations.Add(1)

			iconMutex.Lock()
			if len(iconContents) >= iconCacheCap {
				iconContents = make(map[string]string)
			}
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
		size, size, html.EscapeString(fill), html.EscapeString(stroke), html.EscapeString(strokeWidth), html.EscapeString(props.Class), content), nil
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
