// Package timeline renders vertical dated events: an activity history, a
// change log, a build's event feed. Items carry a time, a title, an optional
// body, meta line, link and icon, and a badge.Variant that tones the dot on
// the rail. Items are sorted by time (newest first unless OldestFirst) and
// grouped under one heading per calendar day (unless NoGroups); the day is
// taken in Props.Location when set, otherwise in each item's own location.
//
// Layout: every row is a two-column grid with a fixed 1rem gutter holding the
// dot and the rail segment, and a min-width-0 content column, so the rail and
// the dots stay aligned however long a title wraps (no floats, no absolute
// positioning).
//
// Phone width (390px): the same layout; the gutter keeps its 1rem width and
// 0.75rem gap, the time sits on its own line above the title, and titles,
// bodies and meta wrap inside the content column. Nothing is hidden.
package timeline

import (
	"sort"
	"time"

	"github.com/a-h/templ"
	"github.com/bplus/webframe/components/badge"
	"github.com/bplus/webframe/components/emptystate"
	"github.com/bplus/webframe/utils"
)

// Item is one dated event. At is required; a zero At sorts first when
// OldestFirst and last otherwise and is grouped under 0001-01-01. Title is
// the bold first line and becomes a link when Href is set. Body is the
// muted text under it; Meta is a small muted line after the body (an actor,
// a reference). Icon, when set, is rendered in the gutter in place of the
// dot, colored by Variant; pass a 16px glyph (icon.Props{Size: 16}).
// Variant tones the dot: the badge variants map to the same tokens the
// badge uses (see Tone); the zero value is the primary dot.
type Item struct {
	At      time.Time
	Title   string
	Body    string
	Icon    templ.Component
	Variant badge.Variant
	Href    string
	Meta    string
}

// Props configures a Timeline. Class is merged via utils.TwMerge onto the
// root <ol>; Attributes spread onto it.
type Props struct {
	ID         string
	Class      string
	Attributes templ.Attributes
	Items      []Item
	// Location, when set, is the zone every item is shown and grouped in.
	// nil uses each item's own location (time.Time.Location).
	Location *time.Location
	// OldestFirst orders items (and day groups) ascending. The zero value
	// shows the newest first.
	OldestFirst bool
	// NoGroups renders one flat list with no day headings. The zero value
	// groups items by calendar day. The HH:MM shown per item does not carry
	// the date; without groups the date is only in the <time datetime>.
	NoGroups bool
	// Empty is the emptystate shown when Items is empty. A zero value gets
	// the title "Nothing here yet".
	Empty emptystate.Props
}

// Group is one calendar day of items, in display order. Exported so an app
// can test or inspect the grouping; Timeline computes it itself.
type Group struct {
	// Day is midnight of the day in the resolved location.
	Day   time.Time
	Items []Item
}

const (
	dayAttr   = "2006-01-02"
	dayText   = "Monday, January 2, 2006"
	clockText = "15:04"
)

// in returns t in Props.Location when set, else t unchanged.
func (p Props) in(t time.Time) time.Time {
	if p.Location != nil {
		return t.In(p.Location)
	}
	return t
}

// sorted returns a copy of Items ordered by At, newest first unless
// OldestFirst. The sort is stable so equal times keep their input order.
func (p Props) sorted() []Item {
	out := make([]Item, len(p.Items))
	copy(out, p.Items)
	sort.SliceStable(out, func(i, j int) bool {
		if p.OldestFirst {
			return out[i].At.Before(out[j].At)
		}
		return out[i].At.After(out[j].At)
	})
	return out
}

// Groups returns the items sorted and grouped by calendar day in the
// resolved location. With NoGroups the result is a single Group whose Day
// is the zero time. Consecutive items sharing a day share a group, so
// groups follow the item order.
//
// @pure
func (p Props) Groups() []Group {
	items := p.sorted()
	if p.NoGroups {
		if len(items) == 0 {
			return nil
		}
		return []Group{{Items: items}}
	}
	var groups []Group
	var lastKey string
	for _, it := range items {
		t := p.in(it.At)
		key := t.Format(dayAttr)
		if len(groups) == 0 || key != lastKey {
			day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
			groups = append(groups, Group{Day: day})
			lastKey = key
		}
		groups[len(groups)-1].Items = append(groups[len(groups)-1].Items, it)
	}
	return groups
}

// dayAttr is the ISO date for the day heading's datetime attribute.
func (g Group) dayAttr() string { return g.Day.Format(dayAttr) }

// dayText is the readable day heading.
func (g Group) dayText() string { return g.Day.Format(dayText) }

// atAttr is the RFC 3339 timestamp for an item's datetime attribute, in the
// resolved location.
func (p Props) atAttr(it Item) string { return p.in(it.At).Format(time.RFC3339) }

// atText is the HH:MM shown for an item, in the resolved location.
func (p Props) atText(it Item) string {
	if p.NoGroups {
		// Without day groups the date has to live on the item itself.
		return p.in(it.At).Format("Jan 2 15:04")
	}
	return p.in(it.At).Format(clockText)
}

// empty resolves the zero-value Empty props to the default title.
func (p Props) empty() emptystate.Props {
	e := p.Empty
	if e.Title == "" && e.Description == "" && e.Icon == "" {
		e.Title = "Nothing here yet"
	}
	return e
}

func (p Props) rootClass() string {
	return utils.TwMerge("flex flex-col", p.Class)
}

// dotClass is the class of the dot in the gutter.
func dotClass(v badge.Variant) string {
	return utils.TwMerge("mt-1.5 size-3 shrink-0 rounded-full", Tone(v))
}

// iconClass is the class of the icon wrapper in the gutter.
func iconClass(v badge.Variant) string {
	return utils.TwMerge("mt-0.5 flex size-4 shrink-0 items-center justify-center [&>svg]:size-4", iconTone(v))
}
