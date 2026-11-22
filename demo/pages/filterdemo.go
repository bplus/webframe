package pages

import (
	"net/url"
	"strings"

	"github.com/bplus/webframe/components/filterbar"
)

// The data behind the showcase's Filter bar section (filterbar.templ): a
// search box and a single-choice group of 16 schools over a fixed list of
// spells.

const (
	// FilterDemoID is the demo bar's id; its results region is
	// FilterDemoID+"-results".
	FilterDemoID = "filter-demo"
	// FilterDemoSearch and FilterDemoSchool are the bar's parameter names.
	// They differ from every other name on the showcase page, whose address
	// bar the bar mirrors them into.
	FilterDemoSearch = "fq"
	FilterDemoSchool = "school"
)

// FilterDemoSpell is one row of the demo's results.
type FilterDemoSpell struct {
	Name, School string
}

// filterDemoSchools are the 16 schools and their spells, in display order.
// A school's parameter value is its name in lower case.
var filterDemoSchools = []struct {
	name   string
	spells []string
}{
	{"Abjuration", []string{"Arcane Ward", "Counterspell", "Shield of Faith", "Stoneskin"}},
	{"Alchemy", []string{"Acid Flask", "Elixir of Vigor"}},
	{"Artifice", []string{"Clockwork Familiar", "Rune of Mending", "Spark Engine"}},
	{"Beast", []string{"Call of the Pack", "Hawk Sight"}},
	{"Blood", []string{"Crimson Pact"}},
	{"Chronomancy", []string{"Borrowed Minute", "Haste", "Slow"}},
	{"Conjuration", []string{"Fog Cloud", "Misty Step", "Summon Familiar", "Web"}},
	{"Divination", []string{"Detect Magic", "Guidance", "True Sight"}},
	{"Enchantment", []string{"Charm Person", "Hold Person", "Sleep"}},
	{"Evocation", []string{"Fire Bolt", "Fireball", "Lightning Bolt", "Magic Missile", "Wall of Fire"}},
	{"Geomancy", []string{"Earthen Grasp", "Stone Wall"}},
	{"Illusion", []string{"Invisibility", "Mirror Image", "Silent Image"}},
	{"Necromancy", []string{"Animate Dead", "Chill Touch", "Ray of Sickness"}},
	{"Runes", []string{"Glyph of Warding", "Rune of Fire"}},
	{"Shadow", []string{"Cloak of Night", "Shadow Step"}},
	{"Transmutation", []string{"Enlarge", "Feather Fall", "Fire Shape", "Polymorph"}},
}

// FilterDemo answers what the demo shows for the query q: the bar's controls
// (the search text; each school with the number of its spells that match the
// search; the chosen school selected), the matching spells of the chosen
// school, or of every school when none is chosen, and the number of spells
// in all.
func FilterDemo(q url.Values) (filters []filterbar.FilterConfig, found []FilterDemoSpell, total int) {
	search := strings.ToLower(strings.TrimSpace(q.Get(FilterDemoSearch)))
	school := q.Get(FilterDemoSchool)
	opts := make([]filterbar.FilterOption, 0, len(filterDemoSchools))
	for _, s := range filterDemoSchools {
		value := strings.ToLower(s.name)
		n := 0
		for _, name := range s.spells {
			total++
			if search != "" && !strings.Contains(strings.ToLower(name), search) {
				continue
			}
			n++
			if school == "" || school == value {
				found = append(found, FilterDemoSpell{Name: name, School: s.name})
			}
		}
		opts = append(opts, filterbar.FilterOption{Label: s.name, Value: value, Count: filterbar.Count(n)})
	}
	filters = filterbar.WithSelected([]filterbar.FilterConfig{
		{ID: FilterDemoSearch, Label: "Search", Placeholder: "Search spells", Type: filterbar.TypeSearch, Grow: true},
		{ID: FilterDemoSchool, Label: "School", Placeholder: "All", Type: filterbar.TypeChoice, Options: opts},
	}, q)
	return filters, found, total
}
