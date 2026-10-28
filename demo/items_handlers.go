package demo

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/bplus/webframe/components/action"
	"github.com/bplus/webframe/components/appshell"
	"github.com/bplus/webframe/components/badge"
	"github.com/bplus/webframe/components/datatable"
	"github.com/bplus/webframe/components/filterbar"
	"github.com/bplus/webframe/components/formfield"
	"github.com/bplus/webframe/components/listpage"
	"github.com/bplus/webframe/components/pageheader"
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
	mux.Handle("POST /items/publish", side(HandleItemsPublish))
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
			{Key: "updated", Label: "Updated", Sortable: true, Mono: true, Shrink: true},
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

// specFromCurrentURL rebuilds the list Spec a mutation (delete, publish)
// should re-render for: from the HX-Current-URL header's query when htmx
// sent one (the page the user is looking at), else the request's own
// query.
func specFromCurrentURL(r *http.Request) listquery.Spec {
	q := r.URL.Query()
	if cur := r.Header.Get("HX-Current-URL"); cur != "" {
		if u, err := url.Parse(cur); err == nil {
			q = u.Query()
		}
	}
	return listquery.ParseValues(q, itemsQuery)
}

// renderResults writes just the results region (table + pagination) for
// spec, the response to a mutation issued from inside the list.
func renderResults(w http.ResponseWriter, r *http.Request, spec listquery.Spec) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := listpage.Results(itemsListProps(spec)).Render(r.Context(), w); err != nil {
		log.Printf("demo: render items results: %v", err)
	}
}

// HandleItemDelete deletes one item (DELETE /items/{id}). From the list
// (HX-Target is the results region) it re-renders the results for the
// current URL so the row disappears and the summary updates; from
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
		renderResults(w, r, specFromCurrentURL(r))
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
	renderResults(w, r, specFromCurrentURL(r))
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
