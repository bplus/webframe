// Package listpage is the declarative list page: a page header, an optional
// intro line, a filter bar, a toolbar with bulk actions and a create button,
// then a dense datatable and pagination inside one swap region. The caller
// declares data (columns, rows, a listquery.Spec, the total) and listpage
// wires sort, filter, search and page changes so each one re-renders only
// the results region, via htmx, and pushes the URL.
//
// Composition order of Page: pageheader → [layout.TwoColumn when Aside is
// set, holding everything below] → Intro <p> → filterbar (only when
// Filters or Search is set) → toolbar (only when Toolbar, Create or
// Selection+BulkActions is set) → Results. Results is the swap region:
// <div id=ResultsID()> datatable + pagination </div>, never a header, so a
// swap never duplicates the title.
//
// Query wiring: every control name goes through Spec.Param, so a prefixed
// Spec ("items" → items.status, items.q, items.sort) stays namespaced.
// Filter controls are named by their FilterConfig.ID (the bare filter key
// listed in listquery.Options.FilterKeys); the search box is "q". The
// filter bar carries sort, dir and per as hidden inputs (never page) so a
// filter change keeps the sort and lands on page 1; "Clear filters" keeps
// them too. Selected state comes from Spec.Filters and Spec.Search.
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
// Phone collapse (below md, 768px): the filter bar wraps; the toolbar
// collapses each side past its first action into a "⋯" menu (toolbar's
// default Overflow); the datatable drops Hidden columns and folds row
// actions into a "⋯" menu; pagination wraps with the summary above the
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
)

const (
	defaultID = "list"
	// searchKey is the search control's bare name; Spec.Param prefixes it.
	searchKey = "q"
)

// Props configures Page and Results. ID defaults to "list" and derives
// every other id: the root <div> is ID, the results region ID+"-results",
// the table ID+"-table", the bulk form ID+"-bulk". Class merges onto the
// root <div> via utils.TwMerge; Attributes spread onto it last.
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

// hasFilterBar reports whether anything would render in the filter bar.
func (p Props) hasFilterBar() bool { return len(p.Filters) > 0 || p.Search != "" }

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
	return configs
}

// hiddenInputs is the list state that rides along with every filter
// change: sort, dir and per (under the Spec's prefix), taken from
// Spec.Values so a page size equal to the parsed default is omitted the
// same way Spec.URL omits it. page, q and filters are never included: a
// filter change lands on page 1 and the controls carry q and the filters.
func (p Props) hiddenInputs() map[string]string {
	vals := p.Spec.Values()
	out := map[string]string{}
	for _, key := range []string{"sort", "dir", "per"} {
		name := p.Spec.Param(key)
		if v := vals.Get(name); v != "" {
			out[name] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// filterbarProps assembles the filter bar for the resolved base path.
func (p Props) filterbarProps(base string) filterbar.Props {
	return filterbar.Props{
		Filters:   p.filterConfigs(),
		HxGet:     base,
		HxTarget:  p.resultsTarget(),
		Hidden:    p.hiddenInputs(),
		ShowReset: true,
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
// "Clear filters" link whose URL is Spec.URL(base, WithoutFilters()) (no
// q, no filters; sort and page size kept) as both href and hx-get, targeting
// the results region with hx-push-url so the address bar resets too; and the
// zero value (datatable renders "Nothing here yet") when not.
func (p Props) empty(base string) emptystate.Props {
	e := p.Empty
	if e.Title != "" || e.Description != "" || e.Icon != "" || e.Action != nil {
		return e
	}
	if p.Spec.HasFilters() {
		clear := p.Spec.URL(base, listquery.WithoutFilters())
		e.Title = "No matches"
		e.Description = "Nothing matches your filters."
		// A plain-link action with the hx-* trio as Attributes renders
		// <a href hx-get hx-target hx-push-url>, the same shape as the sort
		// and page links (an Hx* field would make action.Button drop the
		// href and render a <button>).
		e.Action = &action.Action{
			Label: "Clear filters",
			Href:  clear,
			Attributes: templ.Attributes{
				"hx-get":      clear,
				"hx-target":   p.resultsTarget(),
				"hx-push-url": "true",
			},
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
// receives only Results(p); any other plain htmx request receives the page
// body (Page(p), no shell); a boosted, history-restore or non-htmx request
// receives the whole document. It returns an error naming Middleware when
// no App is in the context, writing nothing. Render errors are returned,
// not written as a 500.
func Respond(w http.ResponseWriter, r *http.Request, title string, p Props) error {
	app, err := appshell.Must(r.Context())
	if err != nil {
		return err
	}
	parts := map[string]templ.Component{p.ResultsID(): Results(p)}
	return app.RenderParts(w, r, title, Page(p), parts)
}
