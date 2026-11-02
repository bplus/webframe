package demo

import (
	"fmt"
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
	"github.com/bplus/webframe/components/descriptionlist"
	"github.com/bplus/webframe/components/detailpage"
	"github.com/bplus/webframe/components/filterbar"
	"github.com/bplus/webframe/components/formfield"
	"github.com/bplus/webframe/components/formpage"
	"github.com/bplus/webframe/components/listpage"
	"github.com/bplus/webframe/components/pageheader"
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
// page, sortable by name/type/status/updated, filterable by type and
// status.
var itemsQuery = listquery.Options{
	DefaultSort:     "updated",
	DefaultDir:      listquery.Desc,
	DefaultPageSize: 25,
	AllowedSorts:    []string{"name", "type", "status", "updated"},
	FilterKeys:      []string{"type", "status"},
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

// RegisterItemRoutes mounts the Items app on mux. Every page handler is
// wrapped in SidebarApp.Middleware so the pages render in the sidebar
// shell (the innermost Middleware wins over the server-wide App), and
// /sidebar redirects to /items.
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
	mux.HandleFunc("GET /sidebar", HandleSidebarRedirect)
}

// HandleSidebarRedirect sends the old sidebar demo URL to the Items list,
// which is now the sidebar-layout page (GET /sidebar → 302 /items).
//
// @boundary HTTP
func HandleSidebarRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/items", http.StatusFound)
}

// ---- list -----------------------------------------------------------------

// HandleItemsList renders the Items list (GET /items): parse the query into
// a Spec, apply it to the store, declare the page, Respond. A sort, filter
// or page request from inside the results region receives only the
// results (listpage.Respond reads HX-Target).
//
// @boundary HTTP
// @side-effects writes HTML response
func HandleItemsList(w http.ResponseWriter, r *http.Request) {
	spec := listquery.Parse(r, itemsQuery)
	if err := listpage.Respond(w, r, "Items", itemsListProps(spec)); err != nil {
		log.Printf("demo: render items: %v", err)
	}
}

// itemsListProps applies spec to the store and declares the list page: the
// sketch's intro line, the Type select and Status pills, the search box,
// the toolbar ("Build mod", "+ Create item", bulk "Publish selected") and
// the five columns with their row actions.
func itemsListProps(spec listquery.Spec) listpage.Props {
	page := listquery.Apply(Items.All(), spec, itemSorters, itemMatch)
	total, published, drafts := Items.Counts()
	now := time.Now()

	rows := make([]datatable.Row, 0, len(page.Items))
	for _, it := range page.Items {
		status := badge.Status(it.Status)
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
				{Text: badge.StatusLabel(it.Status), Badge: &status},
				usedIn,
				{Text: relativeTime(it.Updated, now)},
			},
			Actions: []action.Action{
				{Label: "Edit", Href: "/items/" + it.ID + "/edit"},
				{
					Label:    "Delete",
					HxDelete: "/items/" + it.ID,
					HxTarget: itemsResultsSel,
					Confirm:  &action.Confirm{Message: "Delete this item?"},
				},
			},
		})
	}

	return listpage.Props{
		ID:     itemsListID,
		Header: pageheader.Props{Title: "Items"},
		Intro:  fmt.Sprintf("%s · %s · %s", plural(total, "item"), strconv.Itoa(published)+" published", plural(drafts, "draft")),
		Filters: []filterbar.FilterConfig{
			{ID: "type", Label: "Type", Type: filterbar.TypeSelect, Placeholder: "All types", Options: filterOptions(itemTypeOptions)},
			{ID: "status", Label: "Status", Type: filterbar.TypeToggle, Options: filterOptions(itemStatusOptions)},
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
//
// @pure
func filterOptions(opts []formfield.Option) []filterbar.FilterOption {
	out := make([]filterbar.FilterOption, len(opts))
	for i, o := range opts {
		out[i] = filterbar.FilterOption{Label: o.Label, Value: o.Value}
	}
	return out
}

// plural renders "1 item" / "3 items".
//
// @pure
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
		log.Printf("demo: render items results: %v", err)
	}
}

// HandleItemDelete deletes one item (DELETE /items/{id}). From the list
// (HX-Target is the results region) it re-renders the results for the
// current URL so the row disappears and the summary and intro update; from
// anywhere else over htmx (the detail page) it answers HX-Redirect to
// /items; a plain request gets a 303 to /items. Unknown ids are 404.
//
// @boundary HTTP
// @side-effects mutates Items; writes HTML or a redirect
func HandleItemDelete(w http.ResponseWriter, r *http.Request) {
	if !Items.Delete(r.PathValue("id")) {
		http.NotFound(w, r)
		return
	}
	if r.Header.Get("HX-Target") == itemsResultsID {
		respondResults(w, r)
		return
	}
	redirectTo(w, r, "/items")
}

// HandleItemsPublish publishes the selected items (POST /items/publish,
// repeated "ids" from the bulk form's row checkboxes) and re-renders the
// results for the current URL.
//
// @boundary HTTP
// @side-effects mutates Items; writes HTML response
func HandleItemsPublish(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	Items.Publish(r.Form["ids"])
	respondResults(w, r)
}

// redirectTo sends the browser to location: HX-Redirect with an empty 200
// for an htmx request (htmx ignores a 3xx body and would swap the target
// page in place), 303 See Other otherwise.
func redirectTo(w http.ResponseWriter, r *http.Request, location string) {
	if appshell.IsHX(r) {
		w.Header().Set("HX-Redirect", location)
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, location, http.StatusSeeOther)
}

// ---- detail ---------------------------------------------------------------

// HandleItemDetail renders one item (GET /items/{id}): header with the
// status badge, Back to Items, Edit and Delete; a two-column summary; Notes
// inline and History loaded lazily from /items/{id}/history.
//
// @boundary HTTP
// @side-effects writes HTML response
func HandleItemDetail(w http.ResponseWriter, r *http.Request) {
	it, ok := Items.Get(r.PathValue("id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := detailpage.Respond(w, r, it.Name, itemDetailProps(it)); err != nil {
		log.Printf("demo: render item %s: %v", it.ID, err)
	}
}

// itemDetailProps declares the detail page for it.
func itemDetailProps(it Item) detailpage.Props {
	status := badge.Status(it.Status)
	usedIn := descriptionlist.Item{Label: "Used in", Value: it.UsedIn}
	if it.UsedIn == "" {
		usedIn = descriptionlist.Item{Label: "Used in", Value: "—", Muted: true}
	}
	return detailpage.Props{
		ID: "item",
		Header: pageheader.Props{
			Title:      it.Name,
			Back:       &pageheader.Link{Label: "Items", Href: "/items"},
			Status:     &status,
			StatusText: badge.StatusLabel(it.Status),
			Primary:    &action.Action{Label: "Edit", Href: "/items/" + it.ID + "/edit", Variant: action.VariantPrimary},
			Secondary: []action.Action{{
				Label:    "Delete",
				Variant:  action.VariantDestructive,
				HxDelete: "/items/" + it.ID,
				Confirm:  &action.Confirm{Message: "Delete this item?"},
			}},
		},
		Summary: []descriptionlist.Item{
			{Label: "Type", Value: itemTypeLabel(it.Type)},
			{Label: "Status", Value: badge.StatusLabel(it.Status), Badge: &status},
			usedIn,
			{Label: "Updated", Value: relativeTime(it.Updated, time.Now()) + " ago · " + it.Updated.Format("Jan 2, 2006")},
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
// /items/{id}/history): a description list of three dated lines, written
// as a fragment (the tab swaps it into its panel), not through the shell.
//
// @boundary HTTP
// @side-effects writes HTML response
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
	fragment(w, r, "item history", pages.ItemHistory(lines))
}

// ---- form -----------------------------------------------------------------

// itemFormProps declares the create/edit form page. it is nil for a new
// item. Create and edit share one declaration; only the header, the action
// URL, the Cancel target, the submit label and the validate URL (which
// carries the id so a name check skips the item itself) differ.
func itemFormProps(it *Item) formpage.Props {
	p := formpage.Props{
		ID:       itemFormID,
		Header:   pageheader.Props{Title: "New item", Back: &pageheader.Link{Label: "Items", Href: "/items"}},
		Action:   "/items",
		HxPost:   true,
		Validate: "/items/validate",
		Sections: []formpage.Section{
			{Title: "Details", Fields: []formfield.Field{
				{Name: "name", Label: "Name", Required: true, Placeholder: "Blade of the Fallen Star", Help: "At least 3 characters; names are unique."},
				{Name: "type", Label: "Type", Type: formfield.TypeSelect, Required: true, Options: itemTypeOptions, Placeholder: "Choose a type"},
				{Name: "status", Label: "Status", Type: formfield.TypeRadio, Options: itemStatusOptions, Value: "draft"},
				{Name: "remixable", Label: "Remixable", Type: formfield.TypeToggle, Help: "Let other builders start from this item."},
			}},
			{Title: "Notes", Fields: []formfield.Field{
				{Name: "notes", Label: "Notes", Type: formfield.TypeTextarea, Placeholder: "Where it drops, what it does, what is left to do."},
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

// itemValues is the form's Values for an existing item.
//
// @pure
func itemValues(it Item) map[string]string {
	remixable := ""
	if it.Remixable {
		remixable = "on"
	}
	return map[string]string{
		"name":      it.Name,
		"type":      it.Type,
		"status":    it.Status,
		"remixable": remixable,
		"notes":     it.Notes,
	}
}

// submittedValues reads every declared field from the parsed form, so an
// error re-render preserves exactly what was typed (an unchecked toggle
// comes back as "", which formpage renders unchecked).
func submittedValues(r *http.Request, p formpage.Props) map[string]string {
	vals := make(map[string]string)
	for _, f := range itemFields(p) {
		vals[f.Name] = r.FormValue(f.Name)
	}
	return vals
}

// validateItemField returns the error for one field's value, or "":
// name must be at least 3 characters after trimming and unique among the
// other items (exceptID is the item being edited); type must be one of
// the known types. Other fields have no rules.
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

// applyValues copies validated form values onto it.
//
// @pure
func applyValues(it Item, vals map[string]string) Item {
	it.Name = strings.TrimSpace(vals["name"])
	it.Type = vals["type"]
	it.Status = vals["status"]
	it.Remixable = vals["remixable"] != ""
	it.Notes = strings.TrimSpace(vals["notes"])
	return it
}

// HandleItemNew renders the empty create form (GET /items/new).
//
// @boundary HTTP
// @side-effects writes HTML response
func HandleItemNew(w http.ResponseWriter, r *http.Request) {
	if err := formpage.Respond(w, r, "New item", itemFormProps(nil)); err != nil {
		log.Printf("demo: render new item: %v", err)
	}
}

// HandleItemEdit renders the edit form for one item (GET
// /items/{id}/edit) with its current values.
//
// @boundary HTTP
// @side-effects writes HTML response
func HandleItemEdit(w http.ResponseWriter, r *http.Request) {
	it, ok := Items.Get(r.PathValue("id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := formpage.Respond(w, r, "Edit "+it.Name, itemFormProps(&it)); err != nil {
		log.Printf("demo: render edit item %s: %v", it.ID, err)
	}
}

// HandleItemCreate handles the create form's submit (POST /items). On
// validation errors it re-renders the form with the submitted values and
// the errors (status 200: htmx 2 does not swap a 4xx, so a 422 would leave
// the stale form on screen); on success it saves and redirects to the new
// item's detail page (HX-Redirect for an htmx submit, else 303). It never
// renders a success message.
//
// @boundary HTTP
// @side-effects mutates Items; writes HTML or a redirect
func HandleItemCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	p := itemFormProps(nil)
	vals := submittedValues(r, p)
	if errs := validateItem(vals, ""); len(errs) > 0 {
		p.Values, p.Errors = vals, errs
		if err := formpage.Respond(w, r, "New item", p); err != nil {
			log.Printf("demo: render new item errors: %v", err)
		}
		return
	}
	it := Items.Create(applyValues(Item{}, vals))
	redirectTo(w, r, "/items/"+it.ID)
}

// HandleItemUpdate handles the edit form's submit (POST /items/{id}), with
// the same error and success behavior as HandleItemCreate; the name check
// skips the item itself.
//
// @boundary HTTP
// @side-effects mutates Items; writes HTML or a redirect
func HandleItemUpdate(w http.ResponseWriter, r *http.Request) {
	it, ok := Items.Get(r.PathValue("id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	p := itemFormProps(&it)
	vals := submittedValues(r, p)
	if errs := validateItem(vals, it.ID); len(errs) > 0 {
		p.Values, p.Errors = vals, errs
		if err := formpage.Respond(w, r, "Edit "+it.Name, p); err != nil {
			log.Printf("demo: render edit item %s errors: %v", it.ID, err)
		}
		return
	}
	Items.Update(applyValues(it, vals))
	redirectTo(w, r, "/items/"+it.ID)
}

// HandleItemValidate answers one field's blur validation (POST
// /items/validate[?id=N]). Field.Validate posts only the blurred field
// (hx-params), so the first declared field present in the form is the one
// to validate; the HX-Trigger-Name header htmx also sends is honored first
// when present. ?id names the item being edited
// so its own name is not reported as taken. The answer is the field's
// Feedback slot: a message, or the empty slot when the value is valid.
//
// @boundary HTTP
// @side-effects writes HTML response
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
