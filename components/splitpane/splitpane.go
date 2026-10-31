// Package splitpane is the mail-style two-pane display: a list column on
// the left and the selected item's detail on the right. From md (768px) up
// the two sit side by side and picking an item swaps only the detail
// region; below md the pane shows one of the two, the list or the detail
// with a "Back" link, and the server decides which from the URL. The pane
// is built to hold a listpage.Results (or any list) on the left and a
// detailpage partial (or any component) on the right.
//
// # Routes
//
// The app owns two routes that render the same Props with one difference:
//
//	GET /mail      → Detail nil   (the list alone; the detail region is empty)
//	GET /mail/{id} → Detail set   (the list and the item's detail)
//
// Both handlers build the same List and call Respond. Because the view is
// decided from the URL, a reload lands on the same view, a bookmark works,
// and the browser's Back button works on every width.
//
// # List links
//
// Every link in the list that opens an item carries LinkAttrs(p, href):
//
//	<a { splitpane.LinkAttrs(p, "/mail/42")... }>{ subject }</a>
//
// or, through action, action.Action{Href: "/mail/42", Label: subject,
// Attributes: splitpane.LinkAttrs(p, "/mail/42")}. The attributes are
// href (the plain navigation), hx-get=href, hx-target=DetailTarget(p),
// hx-push-url="true" (the address bar follows the pick) and
// hx-trigger="click[matchMedia('(min-width: 48rem)').matches]".
//
// # Why the trigger filter
//
// From md up a click should swap only the detail region, so the link is an
// htmx request targeting it; Respond answers that request (HX-Target names
// the detail id) with the detail region's content alone. Below md there is
// no detail region to swap into (it is hidden, the list fills the width),
// and the server cannot tell a phone from a desktop by the request. So the
// phone must not send the htmx request at all: the hx-trigger event filter
// makes htmx ignore the click below md, the anchor's default action
// runs, and the browser navigates to href as a plain page load. The server
// then sees /mail/42 without an HX-Target, renders the whole document, and
// the pane shows the detail with the Back link. The filter asks the same
// media query Tailwind's md breakpoint uses (48rem), so the two agree at
// any root font size.
//
// # Phone
//
// Below md the list and the detail stack in the document but only one is
// shown: with Detail nil the list (full width) and an empty, hidden detail
// region; with Detail set the detail, preceded by a ghost "Back" link
// (BackHref, BackLabel), and the list hidden. A list click is a plain
// navigation (a new history entry; Back returns to the list).
//
// # Desktop
//
// From md up the list column is a fixed width tier (Sm 280px, Md 360px,
// Lg 440px), sticky under the shell's sticky chrome and scrolling on its
// own; the detail is the flexible column, showing Placeholder while
// nothing is selected. There is no drag-to-resize.
package splitpane

import (
	"net/http"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/appshell"
)

// Width is the list column's width from md up. The zero value is WidthMd.
type Width string

const (
	// WidthMd is 360px (zero value).
	WidthMd Width = ""
	// WidthSm is 280px.
	WidthSm Width = "sm"
	// WidthLg is 440px.
	WidthLg Width = "lg"
)

// defaultID is the root id when ID is empty; DetailID derives from it.
const defaultID = "splitpane"

// defaultBackLabel is the phone back link's text when BackLabel is empty.
const defaultBackLabel = "Back"

// mdTrigger is the hx-trigger on list links: htmx handles the click only
// from md (768px) up; below that the anchor navigates by itself.
const mdTrigger = "click[matchMedia('(min-width: 48rem)').matches]"

// Props declares the pane. List renders in the list column; Detail, when
// non-nil, in the detail region (nil means no item is selected: the
// region is empty and, on a phone, hidden while the list shows).
// ListWidth is the list column's tier from md up. DetailID is the detail
// region's element id, ID+"-detail" when empty ("splitpane-detail" with
// an empty ID). BackHref is the phone back link's target, normally the
// list route; BackLabel its text ("Back" when empty); the link renders
// only when Detail is set. Title, when set, is the detail region's
// aria-label so assistive tech can name the region. Class is merged onto
// the root with utils.TwMerge; Attributes spread last on the root.
type Props struct {
	ID         string
	Class      string
	Attributes templ.Attributes

	List   templ.Component
	Detail templ.Component

	ListWidth Width

	DetailID string

	BackHref  string
	BackLabel string
	// Placeholder renders in the detail region on md and up while Detail
	// is nil ("Select an item"); never on a phone, where the list shows.
	Placeholder templ.Component

	Title string
}

// id returns ID or the default root id.
func (p Props) id() string {
	if p.ID == "" {
		return defaultID
	}
	return p.ID
}

// detailID returns DetailID or id()+"-detail".
func (p Props) detailID() string {
	if p.DetailID != "" {
		return p.DetailID
	}
	return p.id() + "-detail"
}

// backLabel returns BackLabel or "Back".
func (p Props) backLabel() string {
	if p.BackLabel == "" {
		return defaultBackLabel
	}
	return p.BackLabel
}

// DetailTarget returns the hx-target selector for the detail region:
// "#"+DetailID (with its default applied). LinkAttrs uses it; a caller
// that writes its own htmx attributes points them here.
func DetailTarget(p Props) string {
	return "#" + p.detailID()
}

// LinkAttrs returns the attributes a list link needs so a click swaps the
// detail from md up and navigates below: href (the plain navigation and
// the phone fallback), hx-get=href, hx-target=DetailTarget(p),
// hx-push-url="true" and hx-trigger="click[matchMedia('(min-width: 48rem)').matches]".
// Spread it on the <a> (or pass it as an action.Action's Attributes with
// the same Href). The handler behind href is the detail route, which
// calls Respond; it answers the htmx request with the detail region's
// content and the plain navigation with the whole page. See the package
// doc for why the trigger is filtered.
func LinkAttrs(p Props, href string) templ.Attributes {
	return templ.Attributes{
		"href":        href,
		"hx-get":      href,
		"hx-target":   DetailTarget(p),
		"hx-push-url": "true",
		"hx-trigger":  mdTrigger,
	}
}

// Respond writes the pane for r through the appshell.App installed by
// (*App).Middleware: a plain htmx request whose HX-Target is DetailID
// (a list click from md up) receives only the detail region's content,
// swapped into the region's innerHTML; any other plain htmx request
// receives the pane alone (SplitPane(p), no shell); a boosted,
// history-restore or non-htmx request (a phone's list click, a reload, a
// bookmark, the browser's Back) receives the whole document. It returns
// an error naming Middleware when no App is in the context, writing
// nothing. Render errors are returned, not written as a 500. Both the
// list route and the detail route call it; only Detail differs.
func Respond(w http.ResponseWriter, r *http.Request, title string, p Props) error {
	app, err := appshell.Must(r.Context())
	if err != nil {
		return err
	}
	parts := map[string]templ.Component{p.detailID(): detailContent(p)}
	return app.RenderParts(w, r, title, SplitPane(p), parts)
}
