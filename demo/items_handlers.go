package demo

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bplus/webframe/components/action"
	"github.com/bplus/webframe/components/appshell"
	"github.com/bplus/webframe/components/badge"
	"github.com/bplus/webframe/components/datatable"
	"github.com/bplus/webframe/components/datetimepicker"
	"github.com/bplus/webframe/components/descriptionlist"
	"github.com/bplus/webframe/components/detailpage"
	"github.com/bplus/webframe/components/errorpage"
	"github.com/bplus/webframe/components/filterbar"
	"github.com/bplus/webframe/components/flash"
	"github.com/bplus/webframe/components/formfield"
	"github.com/bplus/webframe/components/formpage"
	"github.com/bplus/webframe/components/jsonviewer"
	"github.com/bplus/webframe/components/listpage"
	"github.com/bplus/webframe/components/logviewer"
	"github.com/bplus/webframe/components/pageheader"
	"github.com/bplus/webframe/components/rating"
	"github.com/bplus/webframe/components/toast"
	"github.com/bplus/webframe/demo/pages"
	"github.com/bplus/webframe/listquery"
)

// The Items app is the demo's acceptance scenario for the page layer: a
// list page (dense table, filters, sort, pagination, row actions, bulk
// publish), a detail page (summary, tabs, lazy history) and a create/edit
// form page (sections, blur validation, error summary), all written as
// page declarations and handlers, with no markup of its own.

const (
	// itemsListID is the listpage ID; its results swap region is
	// itemsResultsID.
	itemsListID     = "items"
	itemsResultsID  = itemsListID + "-results"
	itemsResultsSel = "#" + itemsResultsID
	// itemFormID is the formpage ID (the <form> id and htmx swap target).
	itemFormID = "item-form"
)

// itemsQuery is the list's query contract: newest first by default, 25 per
// page, sortable by name/type/status/updated, filterable by type, status
// and pack.
var itemsQuery = listquery.Options{
	DefaultSort:     "updated",
	DefaultDir:      listquery.Desc,
	DefaultPageSize: 25,
	AllowedSorts:    []string{"name", "type", "status", "updated"},
	FilterKeys:      []string{"type", "status", "pack"},
}

// itemTypeOptions are the Type choices, shared by the filter bar and the
// form's select.
var itemTypeOptions = []formfield.Option{
	{Label: "Weapon", Value: "weapon"},
	{Label: "Armor", Value: "armor"},
	{Label: "Accessory", Value: "accessory"},
}

// itemStatusOptions are the Status choices (the filter pills and the
// form's radio group).
var itemStatusOptions = []formfield.Option{
	{Label: "Published", Value: "published"},
	{Label: "Draft", Value: "draft"},
}

// maxItemTags caps an item's tags: Field.MaxTags on the form's tags field
// (the widget stops at the cap) and the validator (a handcrafted request
// cannot exceed it).
const maxItemTags = 5

// itemTagSuggestions are the tags the form's tags field offers as
// completions (TypeTags renders Options' labels as a datalist); any other
// tag can still be typed.
var itemTagSuggestions = []formfield.Option{
	{Label: "legendary"}, {Label: "rare"}, {Label: "cursed"}, {Label: "fire"}, {Label: "water"},
	{Label: "radiant"}, {Label: "stealth"}, {Label: "two-handed"}, {Label: "ranged"}, {Label: "netherese"},
}

// maxRating is the star count of the rating field and the top of the
// accepted range (0 means unrated).
const maxRating = 5

// Upload limits for the artwork file: the whole request body (every part
// together) and the one file. The filedrop's MaxSize is the same number,
// so the browser refuses first and the server enforces it anyway.
const (
	maxItemFormBytes = 8 << 20
	maxArtworkBytes  = 2 << 20
)

// RegisterItemRoutes mounts the Items app on mux. Every page handler is
// wrapped in SidebarApp.Middleware so the pages render in the sidebar
// shell (the innermost Middleware wins over the server-wide App).
func RegisterItemRoutes(mux *http.ServeMux) {
	side := func(fn http.HandlerFunc) http.Handler { return SidebarApp.Middleware(fn) }
	mux.Handle("GET /items", side(HandleItemsList))
	mux.Handle("GET /items/new", side(HandleItemNew))
	mux.Handle("GET /items/{id}", side(HandleItemDetail))
	mux.Handle("GET /items/{id}/edit", side(HandleItemEdit))
	mux.Handle("GET /items/{id}/history", side(HandleItemHistory))
	mux.Handle("POST /items", side(HandleItemCreate))
	mux.Handle("POST /items/validate", side(HandleItemValidate))
	mux.Handle("POST /items/publish", side(HandleItemsPublish))
	mux.Handle("POST /items/{id}", side(HandleItemUpdate))
	mux.Handle("DELETE /items/{id}", side(HandleItemDelete))
}

// ---- list -----------------------------------------------------------------

// HandleItemsList renders the Items list (GET /items): parse the query into
// a Spec, apply it to the store, declare the page, Respond. A sort, filter
// or page request from inside the results region receives only the
// results (listpage.Respond reads HX-Target).
func HandleItemsList(w http.ResponseWriter, r *http.Request) {
	spec := listquery.Parse(r, itemsQuery)
	if err := listpage.Respond(w, r, "Items", itemsListProps(spec)); err != nil {
		renderFailed(err, "demo: render items")
	}
}

// itemsListProps applies spec to the store and declares the list page: the
// intro line, the Type select and the single-choice Pack buttons with their
// counts, the Status pills (no counts), the search box,
// the toolbar ("Build mod", "+ Create item", bulk "Publish selected") and
// the five columns with their row actions.
func itemsListProps(spec listquery.Spec) listpage.Props {
	all := Items.All()
	page := listquery.Apply(all, spec, itemSorters, itemMatch)
	total, published, drafts := Items.Counts()
	now := time.Now()

	rows := make([]datatable.Row, 0, len(page.Items))
	for _, it := range page.Items {
		usedIn := datatable.Cell{Text: it.UsedIn}
		if it.UsedIn == "" {
			usedIn = datatable.Cell{Text: "—", Muted: true}
		}
		rows = append(rows, datatable.Row{
			ID:   it.ID,
			Href: "/items/" + it.ID,
			Cells: []datatable.Cell{
				{Text: it.Name},
				{Text: itemTypeLabel(it.Type)},
				datatable.BadgeCell(badge.StatusLabel(it.Status), it.Status),
				usedIn,
				{Text: relativeTime(it.Updated, now)},
			},
			Actions: []action.Action{
				{Label: "Edit", Href: "/items/" + it.ID + "/edit"},
				{
					Label:    "Delete",
					Variant:  action.VariantDestructive,
					HxDelete: "/items/" + it.ID,
					HxTarget: itemsResultsSel,
					Confirm:  &action.Confirm{Title: "Delete " + it.Name + "?", Message: "This cannot be undone.", ConfirmLabel: "Delete"},
				},
			},
		})
	}

	return listpage.Props{
		ID:     itemsListID,
		Header: pageheader.Props{Title: "Items"},
		Intro:  fmt.Sprintf("%s · %s · %s", plural(total, "item"), strconv.Itoa(published)+" published", plural(drafts, "draft")),
		Filters: []filterbar.FilterConfig{
			{ID: "type", Label: "Type", Type: filterbar.TypeSelect, Placeholder: "All types", Options: itemTypeFilterOptions(all, spec)},
			{ID: "status", Label: "Status", Type: filterbar.TypeToggle, Options: filterOptions(itemStatusOptions)},
			{ID: "pack", Label: "Pack", Type: filterbar.TypeChoice, Placeholder: "All", Options: itemPackOptions(all, spec)},
		},
		Search:  "Search items",
		Toolbar: []action.Action{{Label: "Build mod", Href: "#"}},
		Create:  &action.Action{Label: "+ Create item", Href: "/items/new"},
		Columns: []datatable.Column{
			{Key: "name", Label: "Name", Sortable: true},
			{Key: "type", Label: "Type", Sortable: true, Hidden: true},
			{Key: "status", Label: "Status", Shrink: true},
			{Key: "used_in", Label: "Used in", Hidden: true},
			{Key: "updated", Label: "Updated", Sortable: true, Mono: true, Shrink: true, Hidden: true},
		},
		Rows:        rows,
		Spec:        page.Spec,
		Total:       page.Total,
		Noun:        "items",
		Selection:   true,
		BulkActions: []action.Action{{Label: "Publish selected", HxPost: "/items/publish"}},
		Dense:       true,
		Base:        "/items",
	}
}

// filterOptions converts form options into filter-bar options (the two
// packages keep separate types; the choices are the same).
func filterOptions(opts []formfield.Option) []filterbar.FilterOption {
	out := make([]filterbar.FilterOption, len(opts))
	for i, o := range opts {
		out[i] = filterbar.FilterOption{Label: o.Label, Value: o.Value}
	}
	return out
}

// itemCounts counts, per value of one filter, the items that choosing that
// value would show: those that match spec's search and its other filters
// (the filter named key itself is left out), grouped by valueOf.
func itemCounts(items []Item, spec listquery.Spec, key string, valueOf func(Item) string) map[string]int {
	others := spec
	others.Filters = make(map[string][]string, len(spec.Filters))
	for k, v := range spec.Filters {
		if k != key {
			others.Filters[k] = v
		}
	}
	counts := map[string]int{}
	for _, it := range items {
		if itemMatch(it, others) {
			counts[valueOf(it)]++
		}
	}
	return counts
}

// itemPackOptions are the Pack choices, each with its count (itemCounts).
func itemPackOptions(items []Item, spec listquery.Spec) []filterbar.FilterOption {
	counts := itemCounts(items, spec, "pack", itemPack)
	out := make([]filterbar.FilterOption, len(itemPacks))
	for i, p := range itemPacks {
		out[i] = filterbar.FilterOption{Label: p.Label, Value: p.Value, Count: filterbar.CountOf(counts[p.Value])}
	}
	return out
}

// itemTypeFilterOptions are the Type select's choices, each with its count
// (itemCounts): "Weapon (5)".
func itemTypeFilterOptions(items []Item, spec listquery.Spec) []filterbar.FilterOption {
	counts := itemCounts(items, spec, "type", func(it Item) string { return it.Type })
	out := filterOptions(itemTypeOptions)
	for i := range out {
		out[i].Count = filterbar.CountOf(counts[out[i].Value])
	}
	return out
}

// plural renders "1 item" / "3 items".
func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return strconv.Itoa(n) + " " + noun + "s"
}

// respondResults answers a mutation issued from inside the list: the list
// state comes from the page the user is looking at (listquery.ParseCurrent
// reads the HX-Current-URL header), and listpage.RespondResults writes the
// results region, the filter bar's hidden state and the intro line out of
// band, and HX-Replace-Url so the address bar follows a clamped page.
func respondResults(w http.ResponseWriter, r *http.Request) {
	spec := listquery.ParseCurrent(r, itemsQuery)
	if err := listpage.RespondResults(w, r, itemsListProps(spec)); err != nil {
		renderFailed(err, "demo: render items results")
	}
}

// protectedItemID is the one item the demo refuses to delete ("1", Blade of
// the Fallen Star, the first seeded row), so the failure path of a real
// action can be seen: the confirm dialog opens, the user confirms, and the
// server says no.
const protectedItemID = "1"

// HandleItemDelete deletes one item (DELETE /items/{id}). The protected
// item is refused (refuseDelete). Otherwise, from the list (HX-Target is
// the results region) it re-renders the results for the current URL so the
// row disappears and the summary and intro update; from anywhere else over
// htmx (the detail page) it answers HX-Redirect to /items; a plain request
// gets a 303 to /items. An unknown id (deleted in another tab, say) gets
// the 404 page: from the list it fills the results region, and from the
// detail page's Delete button, which has no hx-target, errorpage sends it
// to the page content rather than into the button.
func HandleItemDelete(w http.ResponseWriter, r *http.Request) {
	it, ok := Items.Get(r.PathValue("id"))
	if !ok {
		notFound(w, r)
		return
	}
	if it.ID == protectedItemID {
		refuseDelete(w, r, it)
		return
	}
	Items.Delete(it.ID)
	Logs.Append(logviewer.LevelWarn, fmt.Sprintf("deleted item #%s %s", it.ID, it.Name))
	if r.Header.Get("HX-Target") == itemsResultsID {
		respondResults(w, r)
		return
	}
	appshell.Redirect(w, r, "/items")
}

// refuseDelete answers a delete of the protected item: the action failed
// and nothing on the page changed, so the user is told through a toast.
// Over htmx the response is a 409 whose only body is the error toast
// (toast.Respond: HX-Reswap none, so the list stays as it is). A plain request
// has no page for a toast to land on, so it sets an error flash and sends
// the browser back to the item (303); the shell shows the flash once there.
func refuseDelete(w http.ResponseWriter, r *http.Request, it Item) {
	title := "Could not delete " + it.Name
	body := "It is still used in " + it.UsedIn + "; remove it from the pack first."
	if it.UsedIn == "" {
		body = "The demo keeps this item so the refused delete can be seen."
	}
	Logs.Append(logviewer.LevelError, fmt.Sprintf("refused to delete item #%s %s: protected", it.ID, it.Name))
	if appshell.IsHX(r) {
		if err := toast.Respond(w, r, http.StatusConflict, toast.Error(title, body)); err != nil {
			log.Printf("demo: render refused delete toast: %v", err)
		}
		return
	}
	flash.Error(w, r, title, body)
	http.Redirect(w, r, "/items/"+it.ID, http.StatusSeeOther)
}

// notFound answers a page request for an unknown item id with the 404 error
// page, in the sidebar shell (the handler runs under
// SidebarApp.Middleware), so the nav still works.
func notFound(w http.ResponseWriter, r *http.Request) {
	if err := errorpage.NotFound(w, r); err != nil {
		log.Printf("demo: render item not found: %v", err)
	}
}

// HandleItemsPublish publishes the selected items (POST /items/publish,
// repeated "ids" from the bulk form's row checkboxes) and re-renders the
// results for the current URL.
func HandleItemsPublish(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	n := Items.Publish(r.Form["ids"])
	Logs.Append(logviewer.LevelInfo, fmt.Sprintf("published %s (%d selected)", plural(n, "item"), len(r.Form["ids"])))
	respondResults(w, r)
}

// ---- detail ---------------------------------------------------------------

// HandleItemDetail renders one item (GET /items/{id}): header with the
// status badge, Back to Items, Edit and Delete; a two-column summary; Notes
// inline and History loaded lazily from /items/{id}/history. An unknown id
// is the 404 error page.
func HandleItemDetail(w http.ResponseWriter, r *http.Request) {
	it, ok := Items.Get(r.PathValue("id"))
	if !ok {
		notFound(w, r)
		return
	}
	if err := detailpage.Respond(w, r, it.Name, itemDetailProps(it)); err != nil {
		renderFailed(err, "demo: render item %s", it.ID)
	}
}

// itemDetailProps declares the detail page for it: the summary rows plus
// tags (as badges, the rating read-only, the
// release date, the artwork file name), each with a muted dash when empty.
func itemDetailProps(it Item) detailpage.Props {
	usedIn := descriptionlist.Item{Label: "Used in", Value: it.UsedIn}
	if it.UsedIn == "" {
		usedIn = descriptionlist.Item{Label: "Used in", Value: "—", Muted: true}
	}
	tags := descriptionlist.Item{Label: "Tags", Component: pages.TagBadges(it.Tags)}
	if len(it.Tags) == 0 {
		tags = descriptionlist.Item{Label: "Tags", Value: "—", Muted: true}
	}
	stars := descriptionlist.Item{Label: "Rating", Component: rating.Rating(rating.Props{
		ID: "item-rating", ReadOnly: true, Value: it.Rating, Max: maxRating, Size: rating.SizeSm, Label: "Rating",
	})}
	if it.Rating == 0 {
		stars = descriptionlist.Item{Label: "Rating", Value: "Not rated", Muted: true}
	}
	released := descriptionlist.Item{Label: "Released", Value: it.Released.Format("Jan 2, 2006")}
	if it.Released.IsZero() {
		released = descriptionlist.Item{Label: "Released", Value: "—", Muted: true}
	}
	artwork := descriptionlist.Item{Label: "Artwork", Value: it.Artwork, Mono: true}
	if it.Artwork == "" {
		artwork = descriptionlist.Item{Label: "Artwork", Value: "—", Muted: true}
	}
	return detailpage.Props{
		ID: "item",
		Header: pageheader.Props{
			Title:      it.Name,
			Back:       &pageheader.Link{Label: "Items", Href: "/items"},
			Status:     badge.StatusPtr(it.Status),
			StatusText: badge.StatusLabel(it.Status),
			Primary:    &action.Action{Label: "Edit", Href: "/items/" + it.ID + "/edit", Variant: action.VariantPrimary},
			Secondary: []action.Action{{
				Label:    "Delete",
				Variant:  action.VariantDestructive,
				HxDelete: "/items/" + it.ID,
				Confirm:  &action.Confirm{Title: "Delete " + it.Name + "?", Message: "This cannot be undone.", ConfirmLabel: "Delete"},
			}},
		},
		Summary: []descriptionlist.Item{
			{Label: "Type", Value: itemTypeLabel(it.Type)},
			descriptionlist.BadgeItem("Status", badge.StatusLabel(it.Status), it.Status),
			usedIn,
			{Label: "Updated", Value: relativeTime(it.Updated, time.Now()) + " ago · " + it.Updated.Format("Jan 2, 2006")},
			tags,
			stars,
			released,
			artwork,
			{Label: "ID", Value: it.ID, Mono: true, Copyable: true},
		},
		SummaryColumns: 2,
		Tabs: []detailpage.Tab{
			{ID: "notes", Label: "Notes", Content: pages.ItemNotes(it.Notes)},
			{ID: "history", Label: "History", HxGet: "/items/" + it.ID + "/history"},
		},
	}
}

// HandleItemHistory renders the History tab panel (GET
// /items/{id}/history): a description list of three dated lines, a diff of
// the notes against their previous version (demo data: notesDiff) and the
// item's record as a JSON tree, written as a fragment (the tab swaps it
// into its panel), not through the shell.
func HandleItemHistory(w http.ResponseWriter, r *http.Request) {
	it, ok := Items.Get(r.PathValue("id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	date := func(t time.Time) string { return t.Format("Jan 2, 2006") }
	lines := []descriptionlist.Item{
		{Label: date(it.Updated), Value: "Updated", Mono: false},
		{Label: date(it.Updated.Add(-3 * day)), Value: "Status set to " + strings.ToLower(badge.StatusLabel(it.Status))},
		{Label: date(it.Updated.Add(-10 * day)), Value: "Created"},
	}
	fragment(w, r, "item history", pages.ItemHistory(lines, notesDiffProps(it), jsonviewer.Props{
		ID: "item-json", Value: it, Collapsed: 2, CopyPath: true,
	}))
}

// ---- form -----------------------------------------------------------------

// itemFormProps declares the create/edit form page. it is nil for a new
// item. Create and edit share one declaration; only the header, the action
// URL, the Cancel target, the submit label and the validate URL (which
// carries the id so a name check skips the item itself) differ.
func itemFormProps(it *Item) formpage.Props {
	artworkHelp := "PNG, JPEG or SVG up to 2 MB. The demo keeps the file name only."
	if it != nil && it.Artwork != "" {
		artworkHelp = "Current file: " + it.Artwork + ". Choose another to replace it; the demo keeps the file name only."
	}
	p := formpage.Props{
		ID:        itemFormID,
		Header:    pageheader.Props{Title: "New item", Back: &pageheader.Link{Label: "Items", Href: "/items"}},
		Action:    "/items",
		HxPost:    true,
		Multipart: true,
		Validate:  "/items/validate",
		Sections: []formpage.Section{
			{Title: "Details", Fields: []formfield.Field{
				{Name: "name", Label: "Name", Required: true, Placeholder: "Blade of the Fallen Star", Help: "At least 3 characters; names are unique."},
				{Name: "type", Label: "Type", Type: formfield.TypeSelect, Required: true, Options: itemTypeOptions, Placeholder: "Choose a type"},
				{Name: "status", Label: "Status", Type: formfield.TypeRadio, Options: itemStatusOptions, Value: "draft"},
				{Name: "remixable", Label: "Remixable", Type: formfield.TypeToggle, Help: "Let other builders start from this item."},
				{Name: "tags", Label: "Tags", Type: formfield.TypeTags, Options: itemTagSuggestions, MaxTags: maxItemTags, Placeholder: "Add a tag", Help: "Enter or a comma adds a tag, up to five; Backspace removes the last one."},
			}},
			{Title: "Release", Fields: []formfield.Field{
				{Name: "released", Label: "Released", Type: formfield.TypeDate, MaxValue: datetimepicker.Format(datetimepicker.KindDate, time.Now()), Help: "The day the item shipped; not in the future.", Width: formfield.WidthHalf},
				{Name: "rating", Label: "Rating", Type: formfield.TypeRating, Max: maxRating, Help: "Leave every star empty for unrated.", Width: formfield.WidthHalf},
				{Name: "artwork", Label: "Artwork", Type: formfield.TypeFile, Accept: "image/*", Preview: true, Help: artworkHelp},
			}},
			{Title: "Notes", Fields: []formfield.Field{
				{Name: "notes", Label: "Notes", Type: formfield.TypeTextarea, Placeholder: "Where it drops, what it does, what is left to do.", Help: "Plain text; a blank line starts a new paragraph."},
			}},
		},
		Submit: action.Action{Label: "Create item"},
		Cancel: &action.Action{Label: "Cancel", Href: "/items"},
	}
	if it != nil {
		p.Header = pageheader.Props{Title: "Edit item", Subtitle: it.Name, Back: &pageheader.Link{Label: it.Name, Href: "/items/" + it.ID}}
		p.Action = "/items/" + it.ID
		p.Validate = "/items/validate?id=" + url.QueryEscape(it.ID)
		p.Submit = action.Action{Label: "Save"}
		p.Cancel = &action.Action{Label: "Cancel", Href: "/items/" + it.ID}
		p.Values = itemValues(*it)
	}
	return p
}

// itemFields lists every declared field of the form, in order.
func itemFields(p formpage.Props) []formfield.Field {
	var out []formfield.Field
	for _, s := range p.Sections {
		out = append(out, s.Fields...)
	}
	return out
}

// itemValues is the form's Values for an existing item, each in the
// shape its widget submits: tags comma-joined, the release date as the
// native input writes it, the rating as an integer string ("" unrated).
// The artwork is a file input and cannot be refilled, so it has no entry.
func itemValues(it Item) map[string]string {
	remixable := ""
	if it.Remixable {
		remixable = "on"
	}
	stars := ""
	if it.Rating > 0 {
		stars = strconv.Itoa(it.Rating)
	}
	return map[string]string{
		"name":      it.Name,
		"type":      it.Type,
		"status":    it.Status,
		"remixable": remixable,
		"tags":      strings.Join(it.Tags, ","),
		"released":  datetimepicker.Format(datetimepicker.KindDate, it.Released),
		"rating":    stars,
		"notes":     it.Notes,
	}
}

// parseItemForm parses the item form's body: multipart (the form is
// Multipart for the artwork file; the body is capped at maxItemFormBytes)
// or, for a plain urlencoded post, the ordinary form. ParseMultipartForm
// has already read a urlencoded body when it reports ErrNotMultipart, so
// that error is not one.
func parseItemForm(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxItemFormBytes)
	err := r.ParseMultipartForm(maxArtworkBytes)
	if errors.Is(err, http.ErrNotMultipart) {
		return nil
	}
	return err
}

// submittedValues reads every declared field from the parsed form, so an
// error re-render preserves exactly what was typed (an unchecked toggle
// comes back as "", which formpage renders unchecked). The artwork file is
// not a value; readArtwork reads it.
func submittedValues(r *http.Request, p formpage.Props) map[string]string {
	vals := make(map[string]string)
	for _, f := range itemFields(p) {
		vals[f.Name] = r.FormValue(f.Name)
	}
	return vals
}

// readArtwork reads the artwork part of a parsed multipart form and
// returns the file name to store, or the error to show under the field:
// the file must be an image (sniffed from its first bytes, not trusted
// from the browser's type) and at most maxArtworkBytes. No file (the user
// picked none, or the post was not multipart) returns "", "".
func readArtwork(r *http.Request) (name, errMsg string) {
	f, hdr, err := r.FormFile("artwork")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) || errors.Is(err, http.ErrNotMultipart) {
			return "", ""
		}
		return "", "The artwork could not be read; choose it again"
	}
	defer f.Close()
	if hdr.Size > maxArtworkBytes {
		return "", "Artwork must be 2 MB or smaller"
	}
	head := make([]byte, 512)
	n, err := io.ReadFull(f, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return "", "The artwork could not be read; choose it again"
	}
	if n == 0 || !strings.HasPrefix(http.DetectContentType(head[:n]), "image/") {
		return "", "Artwork must be an image (PNG, JPEG, GIF, WebP or SVG)"
	}
	return hdr.Filename, ""
}

// parseTags splits a tags field's value (the tags joined with commas, as
// tagsinput submits them), trims each, drops empties and duplicates
// (compared case-insensitively, the first spelling kept).
func parseTags(value string) []string {
	var out []string
	seen := make(map[string]bool)
	for _, t := range strings.Split(value, ",") {
		t = strings.TrimSpace(t)
		if t == "" || seen[strings.ToLower(t)] {
			continue
		}
		seen[strings.ToLower(t)] = true
		out = append(out, t)
	}
	return out
}

// parseRating reads a rating field's value: "" is 0 (unrated); otherwise
// an integer from 0 to maxRating. Anything else is an error.
func parseRating(value string) (int, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || n < 0 || n > maxRating {
		return 0, fmt.Errorf("rating %q is not 0 to %d", value, maxRating)
	}
	return n, nil
}

// validateItemField returns the error for one field's value, or "":
// name must be at least 3 characters after trimming and unique among the
// other items (exceptID is the item being edited); type must be one of
// the known types; rating must parse to 0–5; released must be a date and
// not after today. Other fields have no rules.
func validateItemField(name, value, exceptID string) string {
	switch name {
	case "name":
		v := strings.TrimSpace(value)
		if v == "" {
			return "Name is required"
		}
		if len([]rune(v)) < 3 {
			return "Name must be at least 3 characters"
		}
		if Items.NameTaken(v, exceptID) {
			return "An item named " + v + " already exists"
		}
	case "type":
		if _, ok := itemTypeLabels[value]; !ok {
			return "Choose a type"
		}
	case "status":
		if value != "published" && value != "draft" {
			return "Choose Published or Draft"
		}
	case "tags":
		if len(parseTags(value)) > maxItemTags {
			return fmt.Sprintf("At most %d tags", maxItemTags)
		}
	case "rating":
		if _, err := parseRating(value); err != nil {
			return fmt.Sprintf("Rating must be 0 to %d", maxRating)
		}
	case "released":
		t, err := datetimepicker.Parse(datetimepicker.KindDate, value, nil)
		if err != nil {
			return "Enter the release date as YYYY-MM-DD"
		}
		if t.After(time.Now()) {
			return "The release date cannot be in the future"
		}
	}
	return ""
}

// validateItem runs validateItemField over every submitted value and
// returns the errors by field name (empty when the form is valid).
func validateItem(vals map[string]string, exceptID string) map[string]string {
	errs := make(map[string]string)
	for name, value := range vals {
		if msg := validateItemField(name, value, exceptID); msg != "" {
			errs[name] = msg
		}
	}
	return errs
}

// applyValues copies validated form values onto it (the artwork name is
// set by the handler from the file part, when one was sent). The rating
// and release date have been validated, so their parse errors are ignored.
func applyValues(it Item, vals map[string]string) Item {
	it.Name = strings.TrimSpace(vals["name"])
	it.Type = vals["type"]
	it.Status = vals["status"]
	it.Remixable = vals["remixable"] != ""
	it.Tags = parseTags(vals["tags"])
	it.Rating, _ = parseRating(vals["rating"])
	it.Released, _ = datetimepicker.Parse(datetimepicker.KindDate, vals["released"], nil)
	it.Notes = strings.TrimSpace(vals["notes"])
	return it
}

// HandleItemNew renders the empty create form (GET /items/new).
func HandleItemNew(w http.ResponseWriter, r *http.Request) {
	if err := formpage.Respond(w, r, "New item", itemFormProps(nil)); err != nil {
		renderFailed(err, "demo: render new item")
	}
}

// HandleItemEdit renders the edit form for one item (GET
// /items/{id}/edit) with its current values. An unknown id is the 404
// error page.
func HandleItemEdit(w http.ResponseWriter, r *http.Request) {
	it, ok := Items.Get(r.PathValue("id"))
	if !ok {
		notFound(w, r)
		return
	}
	if err := formpage.Respond(w, r, "Edit "+it.Name, itemFormProps(&it)); err != nil {
		renderFailed(err, "demo: render edit item %s", it.ID)
	}
}

// HandleItemCreate handles the create form's submit (POST /items,
// multipart or urlencoded). On validation errors it re-renders the form
// with the submitted values and the errors (422; appshell configures htmx
// to swap error responses, so the form re-renders in place; the file
// field comes back empty, as a file input must); on success it saves,
// logs the change and redirects to the new item's detail page (HX-Redirect
// for an htmx submit, else 303) with a "Saved" flash shown once there.
func HandleItemCreate(w http.ResponseWriter, r *http.Request) {
	if err := parseItemForm(w, r); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	p := itemFormProps(nil)
	vals := submittedValues(r, p)
	errs := validateItem(vals, "")
	artwork, artErr := readArtwork(r)
	if artErr != "" {
		errs["artwork"] = artErr
	}
	if len(errs) > 0 {
		p.Values, p.Errors = vals, errs
		if err := formpage.Respond(w, r, "New item", p); err != nil {
			renderFailed(err, "demo: render new item errors")
		}
		return
	}
	it := applyValues(Item{}, vals)
	it.Artwork = artwork
	it = Items.Create(it)
	Logs.Append(logviewer.LevelInfo, fmt.Sprintf("created item #%s %s (%s, %s)", it.ID, it.Name, it.Type, it.Status))
	flash.Success(w, r, "Saved", it.Name+" was created.")
	appshell.Redirect(w, r, "/items/"+it.ID)
}

// HandleItemUpdate handles the edit form's submit (POST /items/{id}), with
// the same error and success behavior as HandleItemCreate; the name check
// skips the item itself, and a submit without a file keeps the stored
// artwork. An unknown id is the 404 error page, like the edit form's own
// GET.
func HandleItemUpdate(w http.ResponseWriter, r *http.Request) {
	it, ok := Items.Get(r.PathValue("id"))
	if !ok {
		notFound(w, r)
		return
	}
	if err := parseItemForm(w, r); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	p := itemFormProps(&it)
	vals := submittedValues(r, p)
	errs := validateItem(vals, it.ID)
	artwork, artErr := readArtwork(r)
	if artErr != "" {
		errs["artwork"] = artErr
	}
	if len(errs) > 0 {
		p.Values, p.Errors = vals, errs
		if err := formpage.Respond(w, r, "Edit "+it.Name, p); err != nil {
			renderFailed(err, "demo: render edit item %s errors", it.ID)
		}
		return
	}
	// No new file keeps the stored artwork: a file input cannot carry the
	// old choice back.
	updated := applyValues(it, vals)
	if artwork != "" {
		updated.Artwork = artwork
	}
	Items.Update(updated)
	Logs.Append(logviewer.LevelInfo, fmt.Sprintf("updated item #%s %s (%s, %s)", updated.ID, updated.Name, updated.Type, updated.Status))
	flash.Success(w, r, "Saved", updated.Name+" was updated.")
	appshell.Redirect(w, r, "/items/"+updated.ID)
}

// HandleItemValidate answers one field's blur validation (POST
// /items/validate[?id=N]). Field.Validate posts only the blurred field
// (hx-params), so the first declared field present in the form is the one
// to validate; the HX-Trigger-Name header htmx also sends is honored first
// when present. ?id names the item being edited
// so its own name is not reported as taken. The answer is the field's
// Feedback slot: a message, or the empty slot when the value is valid.
func HandleItemValidate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	p := itemFormProps(nil)
	field, ok := p.Field(r.Header.Get("HX-Trigger-Name"))
	if !ok {
		for _, f := range itemFields(p) {
			if r.Form.Has(f.Name) {
				field, ok = f, true
				break
			}
		}
	}
	if !ok {
		http.Error(w, "no known field in the request", http.StatusBadRequest)
		return
	}
	msg := validateItemField(field.Name, r.FormValue(field.Name), r.URL.Query().Get("id"))
	if err := formfield.RespondFeedback(w, r, field, msg); err != nil {
		log.Printf("demo: render item validate %s: %v", field.Name, err)
	}
}
