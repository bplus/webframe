package appshell

import (
	"strings"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/action"
)

// Layout selects how Shell arranges navigation.
type Layout string

const (
	// LayoutTopbar renders a horizontal topbar (zero value).
	LayoutTopbar Layout = ""
	// LayoutSidebar renders a left sidebar with grouped links.
	LayoutSidebar Layout = "sidebar"
)

// Width is the max-width of the <main class="wf-content"> region.
type Width string

const (
	// WidthDefault caps content at 1280px (zero value, no modifier class).
	WidthDefault Width = ""
	// WidthNarrow caps content at 768px.
	WidthNarrow Width = "narrow"
	// WidthWide caps content at 1536px.
	WidthWide Width = "wide"
	// WidthFull removes the cap.
	WidthFull Width = "full"
)

// LiveBadge is an HTMX-polled badge rendered after a nav item's label.
// Every is an htmx interval such as "30s"; empty means "30s".
type LiveBadge struct{ HxGet, Every string }

// LiveDot is an HTMX-polled status dot rendered at the start of the
// topbar's right side. Every is an htmx interval such as "30s"; empty
// means "30s".
type LiveDot struct{ HxGet, Every string }

// defaultEvery is the poll interval used when a LiveBadge or LiveDot leaves
// Every empty.
const defaultEvery = "30s"

// everyTrigger returns the htmx "every <interval>" trigger for every,
// defaulting an empty interval to defaultEvery.
func everyTrigger(every string) string {
	if every == "" {
		every = defaultEvery
	}
	return "every " + every
}

// NavItem is one navigation link. Match, when set, marks the item active
// for any path starting with it (e.g. "/factory" for "/factory/123");
// otherwise the item is active only when the path equals Href.
type NavItem struct {
	Label, Href, Match string
	Icon               templ.Component
	Badge              *LiveBadge
}

// NavGroup is a titled group of sidebar links.
type NavGroup struct {
	Title string
	Items []NavItem
}

// User is the signed-in user shown in the topbar's avatar menu.
type User struct {
	Name, AvatarURL string
	Menu            []action.Action
}

// Props configures Shell.
type Props struct {
	ID         string
	Class      string
	Attributes templ.Attributes

	AppName, HomeHref string
	Layout            Layout
	// Nav holds topbar items, or ungrouped sidebar items.
	Nav []NavItem
	// Groups holds titled sidebar groups (sidebar layout only).
	Groups []NavGroup
	// CurrentPath decides which item is active.
	CurrentPath string
	// MaxVisible is how many topbar items show before the "More"
	// disclosure; defaults to 4.
	MaxVisible  int
	User        *User
	Right       templ.Component
	ThemeToggle bool
	LiveDot     *LiveDot
	Width       Width
	// Flash, when non-nil, renders inside the #wf-flash slot at the top
	// of <main>, above the page body; it is the only thing that slot ever
	// holds. (*App).RenderParts fills it from App.Flash.
	Flash templ.Component
}

// phoneVisible is how many topbar items stay inline under 768px; the
// rest are reachable through the More disclosure.
const phoneVisible = 2

// isActive reports whether item is the current page for path: an exact
// Href match, or a Match prefix match.
func isActive(item NavItem, path string) bool {
	if item.Href == path {
		return true
	}
	return item.Match != "" && strings.HasPrefix(path, item.Match)
}

func maxVisible(p Props) int {
	if p.MaxVisible <= 0 {
		return 4
	}
	return p.MaxVisible
}

func homeHref(p Props) string {
	if p.HomeHref == "" {
		return "/"
	}
	return p.HomeHref
}

func contentClass(w Width) string {
	if w == WidthDefault {
		return "wf-content"
	}
	return "wf-content wf-content--" + string(w)
}

// splitNav divides topbar items into those shown inline and those only
// in the More list. Items in inline past phoneVisible are also repeated
// in the More list as phone-only copies (CSS shows exactly one copy).
func splitNav(items []NavItem, max int) (inline, phoneOnly, overflow []NavItem) {
	if len(items) <= max {
		inline = items
	} else {
		inline, overflow = items[:max], items[max:]
	}
	if len(inline) > phoneVisible {
		phoneOnly = inline[phoneVisible:]
	}
	return inline, phoneOnly, overflow
}

// navLinkClass returns base, plus base+"--active" when the item is
// active, plus extra.
func navLinkClass(item NavItem, path, base, extra string) string {
	c := base
	if isActive(item, path) {
		c += " " + base + "--active"
	}
	if extra != "" {
		c += " " + extra
	}
	return c
}

func navLinkAttrs(item NavItem, path string) templ.Attributes {
	if isActive(item, path) {
		return templ.Attributes{"aria-current": "page"}
	}
	return templ.Attributes{}
}

// menuItemAttrs builds the hx-* attributes for a non-link user-menu
// action, merged with its own Attributes. The dropdown item itself is
// the <button>, so action.Button is not nested inside it.
func menuItemAttrs(a action.Action) templ.Attributes {
	attrs := templ.Attributes{}
	set := func(k, v string) {
		if v != "" {
			attrs[k] = v
		}
	}
	set("hx-get", a.HxGet)
	set("hx-post", a.HxPost)
	set("hx-delete", a.HxDelete)
	set("hx-target", a.HxTarget)
	set("hx-swap", a.HxSwap)
	if a.Confirm != nil {
		set("hx-confirm", a.Confirm.Message)
	}
	for k, v := range a.Attributes {
		attrs[k] = v
	}
	return attrs
}

func initials(name string) string {
	fields := strings.Fields(name)
	var b strings.Builder
	for i, f := range fields {
		if i == 2 {
			break
		}
		b.WriteString(strings.ToUpper(string([]rune(f)[:1])))
	}
	return b.String()
}

// sidebarOpenData is the Alpine x-data of the sidebar <details>: open
// follows a live (min-width: 768px) media query, so the sidebar is open on
// desktop even where ::details-content is unsupported, and collapses into
// the Menu disclosure under 768px.
const sidebarOpenData = "{ open: false, init() { const mq = window.matchMedia('(min-width: 768px)'); this.open = mq.matches; mq.addEventListener('change', e => { this.open = e.matches }) } }"
