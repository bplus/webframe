package compose_test

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/bplus/webframe/components/badge"
	"github.com/bplus/webframe/components/card"
	"github.com/bplus/webframe/compose"
)

// With fills a container component with children, Join places components
// one after another, and Text adds escaped text. Here: a card holding a
// badge and a line of text.
func Example() {
	page := compose.With(card.Card(card.Props{ID: "release"}),
		compose.Join(
			compose.With(badge.Badge(badge.Props{ID: "new"}), compose.Text("New")),
			compose.Text(" Ready to ship & tested."),
		),
	)

	var html strings.Builder
	if err := page.Render(context.Background(), &html); err != nil {
		fmt.Println(err)
		return
	}
	// The class lists are long, so this example removes them before printing.
	fmt.Println(regexp.MustCompile(` class="[^"]*"`).ReplaceAllString(html.String(), ""))
	// Output:
	// <div id="release"><span id="new">New</span> Ready to ship &amp; tested.</div>
}
