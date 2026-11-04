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
// The usual list column is List(p, items): a <nav> of links, one per
// ListItem, each carrying href and LinkAttrs(p, href), with the selected
// item marked. An app with its own list markup (a component that renders
// anchors) gives every link that opens an item its href plus
// LinkAttrs(p, href):
//
//	<a href={ templ.URL("/mail/42") } { splitpane.LinkAttrs(p, "/mail/42")... }>{ subject }</a>
//
// or, through action, action.Action{Href: "/mail/42", Label: subject,
// Attributes: splitpane.LinkAttrs(p, "/mail/42")}.
//
// # Why a script decides, not a trigger filter
//
// From md up a click should swap only the detail region; below md there is
// no visible detail region, and the server cannot tell a phone from a
// desktop by the request, so the phone must not send the htmx request at
// all. A filtered hx-trigger cannot do that, because htmx calls
// preventDefault on the anchor before it evaluates the filter, leaving the
// link dead on a phone. Instead the link has no hx-get, and
// static/js/splitpane.js (loaded by appshell.Scripts) takes the click only
// when the md media query matches, calling htmx.ajax with the link as the
// source. Below md the anchor navigates as a plain page load and the server
// renders the whole document with the detail and its Back link.
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
//
// # Errors
//
// When a list click's request fails from md up (the item was deleted, the
// server answers 404 or 500, the network is down), the detail region keeps
// what it showed, the address bar keeps its URL, and a toast tells the
// user what happened. The region carries data-wf-swallow-errors, so
// htmx-errors.js drops the error answer instead of swapping it in; then
// static/js/splitpane.js shows one of two toasts the pane renders as
// hidden <template> elements:
//
//   - 404 Not Found or 410 Gone: Props.Gone, by default "That item is no
//     longer here" / "It may have been deleted." (a warning).
//   - any other error status, or no answer at all: Props.Failed, by
//     default "Could not load that item" / "Try again." (an error).
//
// No app code is needed: the app's detail handler answers an unknown id
// with errorpage.NotFound as usual. Set Gone or Failed to change the
// wording. Set Props.ShowErrors to swap the error page into the region
// instead; a toast then appears only when htmx-errors.js still refuses the
// body (a plain-text 500), so no failure is silent. A phone's click is a
// full page load and shows the error page either way.

package splitpane

import (
	"net/http"
	"strconv"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/appshell"
	"github.com/bplus/webframe/components/badge"
	"github.com/bplus/webframe/components/toast"
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

// Props declares the pane. List renders in the list column; Detail, when
// non-nil, in the detail region (nil means nothing is selected). ListWidth
// is the list column's tier from md up. DetailID defaults to ID+"-detail".
// BackHref and BackLabel ("Back" by default) form the phone back link,
// rendered only when Detail is set. Title, when set, is the detail
// region's aria-label. Class is merged onto the root; Attributes spread
// last on it.
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

	// ShowErrors lets an error answer to a list click (from md up) replace
	// the detail region, so an item that has gone shows the app's 404 page
	// there. By default the region carries data-wf-swallow-errors and
	// htmx-errors.js drops the error: the region keeps what it showed and
	// the Gone or Failed toast tells the user. A phone's click is a full
	// page load either way and shows the error page.
	ShowErrors bool

	// Gone is the toast shown when a list click is answered 404 Not Found
	// or 410 Gone. With no Title and no Body it is DefaultGone.
	Gone toast.Props
	// Failed is the toast shown for any other error answer to a list
	// click, or when the request got no answer. With no Title and no Body
	// it is DefaultFailed.
	Failed toast.Props
}

// DefaultGone is the toast for a list click answered 404 or 410: the item
// was deleted (or moved) after the list was drawn.
var DefaultGone = toast.Props{
	Kind:  toast.KindWarning,
	Title: "That item is no longer here",
	Body:  "It may have been deleted.",
}

// DefaultFailed is the toast for any other failed list click.
var DefaultFailed = toast.Props{
	Kind:  toast.KindError,
	Title: "Could not load that item",
	Body:  "Try again.",
}

// orDefault returns t, or def when t has no Title and no Body.
func orDefault(t, def toast.Props) toast.Props {
	if t.Title == "" && t.Body == "" {
		return def
	}
	return t
}

// gone and failed return the two toasts with their defaults applied.
func (p Props) gone() toast.Props   { return orDefault(p.Gone, DefaultGone) }
func (p Props) failed() toast.Props { return orDefault(p.Failed, DefaultFailed) }

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
// detail from md up and navigates below: data-wf-splitpane-link,
// hx-target=DetailTarget(p), hx-push-url="true" and hx-boost="false". There
// is no hx-get; see the package doc. It does not include href, which the
// caller sets on the <a> (or as the action.Action's Href) so it is never
// emitted twice.
func LinkAttrs(p Props, href string) templ.Attributes {
	return templ.Attributes{
		"data-wf-splitpane-link": "true",
		"hx-target":              DetailTarget(p),
		"hx-push-url":            "true",
		// An app with hx-boost on an ancestor must not boost these: a
		// boosted GET is a full-document render that would land inside the
		// detail region. splitpane.js owns the click.
		"hx-boost": "false",
	}
}

// listItemID is the stable id of the i-th List link (ID-list-i): the oob
// copy a detail response carries keeps the ids, so htmx restores focus to
// the picked link after the swap.
func (p Props) listItemID(i int) string {
	return p.listID() + "-" + strconv.Itoa(i)
}

// ListItem is one entry of List: Href is the item's detail route (the
// link's href and the LinkAttrs target), Title its one-line name (shown in
// font-medium, truncated, with the full text in the link's title
// attribute), Meta a short second line in muted small text (a type, a
// date, a sender); when Badge is set, Meta renders inside a badge.Badge
// with those Props beside the title instead (badge.StatusPtr picks the
// variant from a status word). Selected marks the item the detail region
// shows: aria-current="page" and the bg-muted background.
type ListItem struct {
	Href, Title, Meta string
	Selected          bool
	Badge             *badge.Props
}

// listID is the <ul>'s id, id()+"-list", so an app can target the list
// alone with a swap.
func (p Props) listID() string {
	return p.id() + "-list"
}

// listColID is the id of the list column's inner wrapper, the element a
// detail response replaces out of band.
func (p Props) listColID() string {
	return p.id() + "-listcol"
}

// Respond writes the pane for r through the appshell.App installed by
// (*App).Middleware. An htmx request targeting DetailID (a list click from
// md up) gets only the detail region's content; any other plain htmx
// request gets SplitPane(p) without the shell; boosted, history-restore and
// non-htmx requests get the whole document. It returns an error, writing
// nothing, when no App is in the context. Both the list route and the
// detail route call it; only Detail differs.
func Respond(w http.ResponseWriter, r *http.Request, title string, p Props) error {
	app, err := appshell.Must(r.Context())
	if err != nil {
		return err
	}
	// The detail part also carries the list out of band, so the selected
	// item in the list follows the pick (the caller built p.List for this
	// URL, with Selected on the shown item).
	parts := map[string]templ.Component{p.detailID(): detailPart(p, title)}
	return app.RenderParts(w, r, title, SplitPane(p), parts)
}
