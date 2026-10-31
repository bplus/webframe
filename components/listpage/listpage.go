// Package listpage is the declarative list page: a page header, an optional
// intro line, a filter bar, a toolbar with bulk actions and a create button,
// then a dense datatable and pagination inside one swap region. The caller
// declares data (columns, rows, a listquery.Spec, the total) and listpage
// wires sort, filter, search and page changes so each one re-renders only
// the results region, via htmx, and pushes the URL.
//
// Composition order of Page: pageheader → [layout.TwoColumn when Aside is
// set, holding everything below] → Intro <p> → filterbar (only when
// Filters or Search is set, or a Column is Sortable) → toolbar (only when
// Toolbar, Create or Selection+BulkActions is set) → Results. Results is
// the swap region: <div id=ResultsID()> datatable + pagination </div>,
// never a header, so a swap never duplicates the title.
//
// Query wiring: every control name goes through Spec.Param, so a prefixed
// Spec ("items" → items.status, items.q, items.sort) stays namespaced.
// Filter controls are named by their FilterConfig.ID (the bare filter key
// listed in listquery.Options.FilterKeys); the search box is "q". The
// filter bar (id ID+"-filterbar") carries sort, dir and per (never page)
// so a filter change keeps the sort and lands on page 1; "Clear filters"
// keeps them too. dir and per are hidden inputs; sort is a hidden input
// unless a Column is Sortable, when it is the Sort select described
// below. Selected state comes from Spec.Filters and Spec.Search.
//
// Sort select (phone): a sortable column marked Hidden loses its header,
// the sort link, below md, so when any Column is Sortable the bar's last
// control is a <select> named Spec.Param("sort"), labelled "Sort", with
// two options per sortable column, "{Label} ↑" and "{Label} ↓", both
// value=Key; the ↓ option carries data-tui-filterbar-set-dir="desc" and ↑
// "asc" (under the Spec's prefix), which filterbar.min.js writes into the
// bar's hidden dir input before every update, so choosing an option
// issues the same sort=Key&dir=… request a header click does. The option
// matching Spec.Sort and Spec.Dir is selected; when none does (the sort
// is a column that is not Sortable, or the Spec has no sort) a leading
// "Default order" option (value "", dir "") is. The select has the id the
// hidden sort input had, filterbar.HiddenID(ID+"-filterbar", Spec.Param
// ("sort")), carries filterbar's state marker so "Clear filters" leaves
// it alone, and is the element the out-of-band update after a header
// click replaces, so the headers and the select never disagree. It
// renders only below md (its wrapper is md:hidden; the headers do its job
// on md and up) but is read by the bar's script at every width, so sort
// rides along with filter changes there as before. A list with sortable
// columns and no Filters or Search still gets the bar, holding only the
// select and no "Clear filters" button, and the whole bar is md:hidden
// then.
//
// The URL is the list state. Every results-only response (Respond for a
// request targeting the results region, RespondResults for a mutation)
// carries, after the results, out-of-band copies of the bar's hidden
// inputs (and of the Sort select in the hidden sort input's place) and of
// the intro line (hx-swap-oob, matched by id), so a sort chosen from a
// table header is what the next filter change sends and the intro's
// counts follow a delete. The bar's URL rewrite drops its own parameters
// and page (filterbar.Props.Drop) before writing the current values, so a
// cleared filter leaves the address bar too. The empty state's "Clear
// filters" drives the bar's own reset (window.tui.filterbar.reset) so the
// controls clear along with the results.
//
// Bulk actions: when Selection and BulkActions are set, a form.Form with id
// ID+"-bulk" holds one button per action in the toolbar's middle slot (right
// of the Left group). Row checkboxes carry name="ids", value=Row.ID and
// form=ID+"-bulk", so an htmx request from a bulk button submits the
// selected ids as a repeated "ids" parameter (r.Form["ids"] after
// ParseForm). A bulk action with HxPost/HxDelete/HxGet and no HxTarget is
// pointed at "#"+ResultsID() so the results re-render; a plain Href action
// is a link. Bulk buttons are htmx-driven (type="button"): there is no
// native submit fallback.
//
// Phone collapse (below md, 768px): the filter bar wraps and shows the
// Sort select after the filters (so a Hidden sortable column can still be
// sorted; the select and the visible headers' sort state agree after any
// change, since every results response updates the select); the toolbar
// collapses each side past its first action into a "⋯" menu (toolbar's
// default Overflow) and the bulk buttons wrap onto their own row; the
// datatable drops Hidden columns, keeps the selection column and folds
// row actions into a "⋯" menu; pagination wraps with the summary above the
// Previous/Next pair and hides the numbered list. With Aside set, the aside
// stacks under the results (layout.TwoColumn's md:flex-row).
//
// Nothing in Results changes height with page state except the rows: the
// table header stays when there are no rows (the empty state fills one
// cell) and the pagination summary line is always present.
package listpage

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/bplus/webframe/components/action"
	"github.com/bplus/webframe/components/appshell"
	"github.com/bplus/webframe/components/datatable"
	"github.com/bplus/webframe/components/emptystate"
	"github.com/bplus/webframe/components/filterbar"
	"github.com/bplus/webframe/components/pageheader"
	"github.com/bplus/webframe/components/pagination"
	"github.com/bplus/webframe/listquery"
	"github.com/bplus/webframe/utils"
)

const (
	defaultID = "list"
	// searchKey is the search control's bare name; Spec.Param prefixes it.
	searchKey = "q"
	// sortKey and dirKey are the sort state's bare names (the Sort select
	// and the hidden input it sets); Spec.Param prefixes them.
	sortKey = "sort"
	dirKey  = "dir"
	// setPrefix is the filterbar.min.js option attribute that writes a
	// value into a named control when the option is selected.
	setPrefix = "data-tui-filterbar-set-"
	// defaultOrderLabel is the Sort select's option when no sortable column
	// is the current sort.
	defaultOrderLabel = "Default order"
	// introClass styles the intro <p> in Page and its out-of-band copy
	// alike, so the replacement does not move anything.
	introClass = "text-sm text-muted-foreground"
)

// Props configures Page and Results. ID defaults to "list" and derives
// every other id: the root <div> is ID, the results region ID+"-results",
// the table ID+"-table", the bulk form ID+"-bulk", the filter bar
// ID+"-filterbar" (its hidden inputs filterbar.HiddenID of that), the
// intro ID+"-intro". Class merges onto the root <div> via utils.TwMerge;
// Attributes spread onto it last.
type Props struct {
	ID         string
	Class      string
	Attributes templ.Attributes

	// Header is the page header (title, breadcrumbs, header actions).
	Header pageheader.Props
	// Intro is an optional one-line <p> under the header ("14 items · 11
	// published · 3 drafts"). "" renders nothing.
	Intro string

	// Filters are the filter bar's controls. FilterConfig.ID is the bare
	// filter key (as in listquery.Options.FilterKeys); the rendered control
	// name is Spec.Param(ID). Selected/Value state is applied from Spec
	// automatically; any Selected set by the caller is overridden.
	Filters []filterbar.FilterConfig
	// Search is the search box placeholder; "" renders no search box. The
	// box is a TypeSearch control named Spec.Param("q"), labelled "Search",
	// rendered before Filters, with its value from Spec.Search.
	Search string

	// Toolbar actions render in the toolbar's left group.
	Toolbar []action.Action
	// Create renders alone in the toolbar's right group; a zero Variant
	// becomes VariantPrimary.
	Create *action.Action

	Columns []datatable.Column
	Rows    []datatable.Row
	// Spec is the current list state (from listquery.Parse, or the Spec
	// listquery.Apply returned, whose Page is clamped). It drives sort
	// links, page links, filter selection and the hidden sort inputs.
	Spec listquery.Spec
	// Total is the row count across every page, for the pagination
	// summary.
	Total int
	// Noun names the rows in the pagination summary ("items").
	Noun string
	// Empty is shown in one full-width table cell when Rows is empty. The
	// zero value reads "No matches" with a filters hint when
	// Spec.HasFilters(), otherwise datatable's "Nothing here yet".
	Empty emptystate.Props

	// Selection adds the checkbox column; row boxes submit with the bulk
	// form as "ids".
	Selection bool
	// BulkActions render inside form.Form{ID: ID+"-bulk"} in the toolbar
	// when Selection is set; ignored otherwise (there is nothing to select).
	BulkActions []action.Action
	// Dense tightens the table's cell padding.
	Dense bool
	// Base is the URL every sort, page and filter request is built on
	// ("/items"). "" uses the request path stored by appshell.Middleware.
	Base string
	// Aside, when non-nil, renders in a layout.TwoColumn right rail beside
	// everything under the header.
	Aside templ.Component
}

// id resolves the ID default.
func (p Props) id() string {
	if p.ID == "" {
		return defaultID
	}
	return p.ID
}

// ResultsID is the id of the swap region Results renders: ID+"-results"
// ("list-results" by default). Handlers compare it with the HX-Target
// header; Respond does so through appshell.RenderParts.
func (p Props) ResultsID() string { return p.id() + "-results" }

// resultsTarget is the hx-target selector for the results region.
func (p Props) resultsTarget() string { return "#" + p.ResultsID() }

// tableID is the datatable's id.
func (p Props) tableID() string { return p.id() + "-table" }

// bulkFormID is the bulk-actions form id and the row checkboxes' form
// attribute.
func (p Props) bulkFormID() string { return p.id() + "-bulk" }

// filterbarID is the filter bar's id; its hidden inputs are
// filterbar.HiddenID(filterbarID, name).
func (p Props) filterbarID() string { return p.id() + "-filterbar" }

// introID is the intro <p>'s id, the target of its out-of-band copy.
func (p Props) introID() string { return p.id() + "-intro" }

// hasFilterControls reports whether the bar holds a filter or search
// control: what "Clear filters" clears, and what makes the bar worth
// showing on md and up.
func (p Props) hasFilterControls() bool { return len(p.Filters) > 0 || p.Search != "" }

// sortableColumns returns the Columns with Sortable set and a Key, in
// order: the Sort select's choices.
func (p Props) sortableColumns() []datatable.Column {
	var cols []datatable.Column
	for _, c := range p.Columns {
		if c.Sortable && c.Key != "" {
			cols = append(cols, c)
		}
	}
	return cols
}

// hasSortControl reports whether the bar holds the Sort select: when any
// column is sortable (see the package doc).
func (p Props) hasSortControl() bool { return len(p.sortableColumns()) > 0 }

// hasFilterBar reports whether anything would render in the filter bar.
func (p Props) hasFilterBar() bool { return p.hasFilterControls() || p.hasSortControl() }

// sortConfig builds the Sort select (see the package doc): named
// Spec.Param("sort"), two options per sortable column whose
// data-tui-filterbar-set-<dir param> attribute sets the hidden dir input,
// the option matching Spec.Sort and Spec.Dir selected, and a leading
// "Default order" option (value "", dir "") when none matches, so the
// select never claims a sort the page is not in. A zero Spec.Dir counts
// as ascending, as listquery sorts it. PhoneOnly: the headers sort on md
// and up.
func (p Props) sortConfig() filterbar.FilterConfig {
	cols := p.sortableColumns()
	setDir := setPrefix + p.Spec.Param(dirKey)
	dir := p.Spec.Dir
	if dir != listquery.Desc {
		dir = listquery.Asc
	}
	opts := make([]filterbar.FilterOption, 0, 2*len(cols)+1)
	matched := false
	for _, c := range cols {
		for _, d := range []listquery.Dir{listquery.Asc, listquery.Desc} {
			sel := p.Spec.Sort == c.Key && dir == d
			matched = matched || sel
			opts = append(opts, filterbar.FilterOption{
				Label:      c.Label + " " + utils.IfElse(d == listquery.Asc, "↑", "↓"),
				Value:      c.Key,
				Selected:   sel,
				Attributes: templ.Attributes{setDir: string(d)},
			})
		}
	}
	if !matched {
		opts = append([]filterbar.FilterOption{{
			Label:      defaultOrderLabel,
			Value:      "",
			Selected:   true,
			Attributes: templ.Attributes{setDir: ""},
		}}, opts...)
	}
	return filterbar.FilterConfig{
		ID:        p.Spec.Param(sortKey),
		Label:     "Sort",
		Type:      filterbar.TypeSelect,
		Options:   opts,
		PhoneOnly: true,
	}
}

// sortControls is sortConfig as the slice filterbar.HiddenOOB takes: one
// config when the bar has the Sort select, nil otherwise.
func (p Props) sortControls() []filterbar.FilterConfig {
	if !p.hasSortControl() {
		return nil
	}
	return []filterbar.FilterConfig{p.sortConfig()}
}

// hasBulk reports whether the bulk form renders.
func (p Props) hasBulk() bool { return p.Selection && len(p.BulkActions) > 0 }

// hasToolbar reports whether anything would render in the toolbar.
func (p Props) hasToolbar() bool {
	return len(p.Toolbar) > 0 || p.Create != nil || p.hasBulk()
}

// base resolves Base: the caller's value, else the request path stored by
// appshell.Middleware, else "" (query-only URLs relative to the page).
func (p Props) base(path string) string {
	if p.Base != "" {
		return p.Base
	}
	return path
}

// filterConfigs builds the filter bar's controls from Search and Filters:
// selection applied from Spec, then every control renamed through
// Spec.Param so a prefixed Spec reads them back.
func (p Props) filterConfigs() []filterbar.FilterConfig {
	configs := make([]filterbar.FilterConfig, 0, len(p.Filters)+1)
	if p.Search != "" {
		configs = append(configs, filterbar.FilterConfig{
			ID:          searchKey,
			Label:       "Search",
			Placeholder: p.Search,
			Type:        filterbar.TypeSearch,
		})
	}
	configs = append(configs, p.Filters...)

	values := make(map[string][]string, len(p.Spec.Filters)+1)
	for k, v := range p.Spec.Filters {
		values[k] = v
	}
	if p.Spec.Search != "" {
		values[searchKey] = []string{p.Spec.Search}
	}
	configs = filterbar.WithSelected(configs, values)
	for i := range configs {
		configs[i].ID = p.Spec.Param(configs[i].ID)
	}
	// The Sort select comes after WithSelected: its two options per column
	// share a value, so it marks its own selection from Spec.Sort and Dir.
	return append(configs, p.sortControls()...)
}

// hiddenInputs is the list state that rides along with every filter
// change: sort, dir and per (under the Spec's prefix), with the values
// Spec.Values writes, so per is "" when the page size equals the parsed
// default, the same way Spec.URL omits it. All three keys are always
// present, empty or not: filterbar.min.js skips an empty control, and the
// out-of-band copy a results response carries (resultsPart) replaces each
// input by id, which needs an element to replace even when the page
// loaded with no sort at all. page, q and filters are never included: a
// filter change lands on page 1 and the controls carry q and the filters.
// When the bar has the Sort select (a Sortable column), the sort key is
// still present so filterbar promotes the select in the hidden input's
// place (filterbar.Props.Hidden); the map value for sort is then ignored
// and the select's own selected option carries the sort.
func (p Props) hiddenInputs() map[string]string {
	vals := p.Spec.Values()
	out := make(map[string]string, 3)
	for _, key := range []string{"sort", "dir", "per"} {
		name := p.Spec.Param(key)
		out[name] = vals.Get(name)
	}
	return out
}

// filterbarProps assembles the filter bar for the resolved base path: the
// bar's id (so its hidden inputs have ids), the controls, the hidden list
// state, and page as the parameter the bar's URL rewrite drops (a filter
// change lands on page 1). "Clear filters" renders only when there is a
// filter or search control to clear; a bar holding only the Sort select
// is md:hidden as a whole (the headers sort on md and up).
func (p Props) filterbarProps(base string) filterbar.Props {
	return filterbar.Props{
		ID:        p.filterbarID(),
		Class:     utils.If(!p.hasFilterControls(), "md:hidden"),
		Filters:   p.filterConfigs(),
		HxGet:     base,
		HxTarget:  p.resultsTarget(),
		Hidden:    p.hiddenInputs(),
		Drop:      []string{p.Spec.Param("page")},
		ShowReset: p.hasFilterControls(),
	}
}

// createActions is the toolbar's right group: Create alone, primary by
// default.
func (p Props) createActions() []action.Action {
	if p.Create == nil {
		return nil
	}
	a := *p.Create
	if a.Variant == "" {
		a.Variant = action.VariantPrimary
	}
	return []action.Action{a}
}

// bulkAction fills a bulk action's default target: any htmx request with
// no HxTarget re-renders the results region.
func (p Props) bulkAction(a action.Action) action.Action {
	if a.HxTarget == "" && (a.HxPost != "" || a.HxDelete != "" || a.HxGet != "") {
		a.HxTarget = p.resultsTarget()
	}
	return a
}

// empty resolves the Empty default: the caller's props when any field is
// set; otherwise, when filters are active, "No matches" with a hint and a
// "Clear filters" link whose href is Spec.URL(base, WithoutFilters()) (no
// q, no filters; sort and page size kept); and the zero value (datatable
// renders "Nothing here yet") when not.
//
// With filter controls on the page the link's click (Alpine, x-on:click
// .prevent) calls window.tui.filterbar.reset on the bar, which clears the
// bar's controls, keeps its hidden inputs and the Sort select, re-requests
// the results region over htmx and rewrites the URL: the bar then shows
// what the results show. The href stays as the no-JS fallback. The link
// carries no hx-get then, since the reset issues the request and a second
// one from the link would fetch the same results twice. Without filter
// controls (filters came from the URL alone; a bar holding only the Sort
// select owns none of them, so its reset could not clear them from the
// URL) the link is a plain htmx link like the sort and page links: hx-get
// to the same URL, targeting the results with hx-push-url.
func (p Props) empty(base string) emptystate.Props {
	e := p.Empty
	if e.Title != "" || e.Description != "" || e.Icon != "" || e.Action != nil {
		return e
	}
	if !p.Spec.HasFilters() {
		return e
	}
	clear := p.Spec.URL(base, listquery.WithoutFilters())
	e.Title = "No matches"
	e.Description = "Nothing matches your filters."
	// A plain-link action with Attributes renders <a href …>; an Hx* field
	// would make action.Button drop the href and render a <button>.
	e.Action = &action.Action{Label: "Clear filters", Href: clear}
	if p.hasFilterControls() {
		// x-data makes the link an Alpine root, so the handler is live on
		// the first page load too (Alpine starts from [x-data] elements;
		// swapped-in content is initialized by the shell's afterSwap hook).
		e.Action.Attributes = templ.Attributes{
			"x-data":             true,
			"x-on:click.prevent": "window.tui.filterbar.reset(document.getElementById('" + utils.EscapeJS(p.filterbarID()) + "'))",
		}
	} else {
		e.Action.Attributes = templ.Attributes{
			"hx-get":      clear,
			"hx-target":   p.resultsTarget(),
			"hx-push-url": "true",
		}
	}
	return e
}

// tableProps assembles the datatable from the list props for the resolved
// base path (used only for the empty state's Clear filters link; datatable
// resolves Base itself for the sort links).
func (p Props) tableProps(base string) datatable.Props {
	return datatable.Props{
		ID:            p.tableID(),
		Columns:       p.Columns,
		Rows:          p.Rows,
		Spec:          p.Spec,
		Base:          p.Base,
		HxTarget:      p.resultsTarget(),
		Selection:     p.Selection,
		SelectionName: "ids",
		SelectionForm: p.bulkFormID(),
		Dense:         p.Dense,
		Empty:         p.empty(base),
	}
}

// paginationProps assembles the pagination from the list props.
func (p Props) paginationProps() pagination.Props {
	return pagination.Props{
		Total:    p.Total,
		Spec:     p.Spec,
		Base:     p.Base,
		HxTarget: p.resultsTarget(),
		Noun:     p.Noun,
	}
}

// Respond writes the list page for r through the appshell.App installed by
// (*App).Middleware: a plain htmx request whose HX-Target is ResultsID()
// receives only the results part (Results(p) followed by the out-of-band
// copies of the filter bar's hidden inputs and the intro, see the package
// doc); any other plain htmx request receives the page body (Page(p), no
// shell); a boosted, history-restore or non-htmx request receives the
// whole document. It returns an error naming Middleware when no App is in
// the context, writing nothing. Render errors are returned, not written as
// a 500.
func Respond(w http.ResponseWriter, r *http.Request, title string, p Props) error {
	app, err := appshell.Must(r.Context())
	if err != nil {
		return err
	}
	parts := map[string]templ.Component{p.ResultsID(): resultsPart(p)}
	return app.RenderParts(w, r, title, Page(p), parts)
}

// RespondResults writes only the results part for r (Results(p) followed
// by the out-of-band copies of the filter bar's hidden inputs and the
// intro): the response to a mutation issued from inside the list (a row's
// DELETE, a bulk POST) after the handler re-applied the Spec it got from
// listquery.ParseCurrent. It sets Content-Type and HX-Replace-Url to
// p.Spec.URL(base), so the address bar follows the state actually served
// (the page listquery.Apply clamped to after the last row of a page was
// deleted). base is p.Base, else the path of listquery.CurrentURL(r): the
// page the user is looking at, not the mutation's own path; it also
// becomes Base for the sort and page links rendered. No appshell.App is
// needed. Render errors are returned, not written as a 500.
func RespondResults(w http.ResponseWriter, r *http.Request, p Props) error {
	if p.Base == "" {
		p.Base = listquery.CurrentURL(r).Path
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("HX-Replace-Url", p.Spec.URL(p.Base))
	return resultsPart(p).Render(r.Context(), w)
}
