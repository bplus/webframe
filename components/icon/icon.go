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

// iconContents caches generated SVG strings keyed by name and props.
// Guarded by iconMutex; bounded by iconCacheCap.
var (
	iconContents = make(map[string]string)
	iconMutex    sync.RWMutex
)

// iconGenerations counts cache misses, so tests can check that a repeated
// render is served from cache.
var iconGenerations atomic.Int64

// Props configures an icon. The zero value renders the Lucide defaults:
// size 24, fill "none", stroke Color or else "currentColor", stroke width
// "2". Strings are HTML-escaped into their attributes but not otherwise
// validated.
type Props struct {
	Size        int
	Color       string
	Fill        string
	Stroke      string
	StrokeWidth string // e.g. "2.5"
	// Class is written as-is (escaped) into the SVG's class attribute.
	// Unlike most components it is not passed through TwMerge.
	Class string
}

// Icon returns a component constructor for the named Lucide icon; the
// generated icon_defs.go predefines every icon as `var X = Icon("x")`.
// Rendered SVGs are cached per name and props. An unknown name is an error
// at render time, not here.
func Icon(name string) func(...Props) templ.Component {
	return func(props ...Props) templ.Component {
		var p Props
		if len(props) > 0 {
			p = props[0]
		}

		// Values are quoted so the key is unambiguous: a value containing
		// "|f:" cannot spell another Props' key.
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

			// Two goroutines may both generate an uncached icon; the
			// writes are identical, so the race is benign.
			generatedSvg, err := generateSVG(name, p)
			if err != nil {
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

// generateSVG builds the <svg> for name with props' defaults applied. It
// fails only when name is not in internalSvgData.
func generateSVG(name string, props Props) (string, error) {
	content, err := getIconContent(name)
	if err != nil {
		return "", err
	}

	size := props.Size
	if size <= 0 {
		size = 24
	}

	fill := props.Fill
	if fill == "" {
		fill = "none"
	}

	stroke := props.Stroke
	if stroke == "" {
		stroke = props.Color
	}
	if stroke == "" {
		stroke = "currentColor"
	}

	strokeWidth := props.StrokeWidth
	if strokeWidth == "" {
		strokeWidth = "2"
	}

	return fmt.Sprintf("<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"%d\" height=\"%d\" viewBox=\"0 0 24 24\" fill=\"%s\" stroke=\"%s\" stroke-width=\"%s\" stroke-linecap=\"round\" stroke-linejoin=\"round\" class=\"%s\" data-lucide=\"icon\">%s</svg>",
		size, size, html.EscapeString(fill), html.EscapeString(stroke), html.EscapeString(strokeWidth), html.EscapeString(props.Class), content), nil
}

// getIconContent returns the inner SVG markup for name from the generated
// internalSvgData.
func getIconContent(name string) (string, error) {
	content, exists := internalSvgData[name]
	if !exists {
		return "", fmt.Errorf("icon '%s' not found in internalSvgData map", name)
	}
	return content, nil
}
