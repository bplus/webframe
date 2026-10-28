package demo

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bplus/webframe/listquery"
)

// Item is one record of the demo's "Items" app (a dashboard
// sketch): a mod item with a type, a publish status, the mod pack it is
// used in, its last update and free-text notes.
type Item struct {
	ID        string
	Name      string
	Type      string // weapon, armor, accessory
	Status    string // published, draft
	UsedIn    string // "" when the item is in no pack
	Updated   time.Time
	Notes     string
	Remixable bool
}

// itemTypeLabels maps a type value to its display label.
var itemTypeLabels = map[string]string{
	"weapon":    "Weapon",
	"armor":     "Armor",
	"accessory": "Accessory",
}

// itemTypeLabel returns the display label for a type value, or the value
// itself when it is not a known type.
//
// @pure
func itemTypeLabel(t string) string {
	if l, ok := itemTypeLabels[t]; ok {
		return l
	}
	return t
}

// ItemStore is the demo's in-memory item store. It is safe for concurrent
// use; every accessor copies, so callers never see later mutations.
type ItemStore struct {
	mu    sync.RWMutex
	items []Item
	next  int
}

// seedItems are the 14 items of the dashboard sketch; Updated is an age
// relative to the store's creation time.
var seedItems = []struct {
	name, typ, status, usedIn string
	age                       time.Duration
}{
	{"Blade of the Fallen Star", "weapon", "published", "Starfall Pack", 2 * day},
	{"Cloak of Whispers", "armor", "draft", "", 3 * day},
	{"Ring of Hoarded Light", "accessory", "published", "Starfall Pack", 5 * day},
	{"Netherese Lesser Folly", "weapon", "published", "Netherese Relics", 1 * week},
	{"Gloves of the Tidecaller", "armor", "published", "", 1 * week},
	{"Amulet of Still Water", "accessory", "draft", "", 2 * week},
	{"Hollowed Greatbow", "weapon", "published", "Starfall Pack", 2 * week},
	{"Boots of Quiet Steps", "armor", "published", "Netherese Relics", 3 * week},
	{"Circlet of Borrowed Thought", "accessory", "published", "", 3 * week},
	{"Ashen Dagger", "weapon", "published", "Netherese Relics", 1 * month},
	{"Helm of the Last Watch", "armor", "draft", "", 1 * month},
	{"Pendant of the Weave", "accessory", "published", "Starfall Pack", 1 * month},
	{"Stormsinger Lute", "weapon", "published", "", 2 * month},
	{"Mantle of Ember", "armor", "published", "Netherese Relics", 2 * month},
}

const (
	day   = 24 * time.Hour
	week  = 7 * day
	month = 30 * day
)

// NewItemStore returns a store seeded with the sketch's 14 items, ids "1"
// to "14" in sketch order, with Updated set relative to now.
func NewItemStore(now time.Time) *ItemStore {
	s := &ItemStore{next: 1}
	for _, seed := range seedItems {
		s.items = append(s.items, Item{
			ID:      strconv.Itoa(s.next),
			Name:    seed.name,
			Type:    seed.typ,
			Status:  seed.status,
			UsedIn:  seed.usedIn,
			Updated: now.Add(-seed.age),
			Notes:   "Seeded from the 2026-10-21 dashboard sketch.",
		})
		s.next++
	}
	return s
}

// Items is the demo server's store. Tests replace it with a fresh one.
var Items = NewItemStore(time.Now())

// All returns a copy of every item in insertion order.
func (s *ItemStore) All() []Item {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Item, len(s.items))
	copy(out, s.items)
	return out
}

// Get returns the item with id, if any.
func (s *ItemStore) Get(id string) (Item, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, it := range s.items {
		if it.ID == id {
			return it, true
		}
	}
	return Item{}, false
}

// NameTaken reports whether another item (one whose id is not exceptID)
// already has this name, compared case-insensitively after trimming.
func (s *ItemStore) NameTaken(name, exceptID string) bool {
	want := strings.ToLower(strings.TrimSpace(name))
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, it := range s.items {
		if it.ID != exceptID && strings.ToLower(it.Name) == want {
			return true
		}
	}
	return false
}

// Create stores it under a fresh id with Updated set to now and returns
// the stored item.
func (s *ItemStore) Create(it Item) Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	it.ID = strconv.Itoa(s.next)
	s.next++
	it.Updated = time.Now()
	s.items = append(s.items, it)
	return it
}

// Update replaces the item with it.ID, setting Updated to now. It reports
// false when no item has that id.
func (s *ItemStore) Update(it Item) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.items {
		if s.items[i].ID == it.ID {
			it.Updated = time.Now()
			s.items[i] = it
			return true
		}
	}
	return false
}

// Delete removes the item with id and reports whether it existed.
func (s *ItemStore) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.items {
		if s.items[i].ID == id {
			s.items = append(s.items[:i], s.items[i+1:]...)
			return true
		}
	}
	return false
}

// Publish marks every item whose id is in ids as published (touching
// Updated) and returns how many changed status.
func (s *ItemStore) Publish(ids []string) int {
	want := make(map[string]bool, len(ids))
	for _, id := range ids {
		want[id] = true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for i := range s.items {
		if want[s.items[i].ID] && s.items[i].Status != "published" {
			s.items[i].Status = "published"
			s.items[i].Updated = time.Now()
			n++
		}
	}
	return n
}

// Counts returns the total number of items and how many are published and
// draft, for the list page's intro line.
func (s *ItemStore) Counts() (total, published, drafts int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, it := range s.items {
		total++
		switch it.Status {
		case "published":
			published++
		case "draft":
			drafts++
		}
	}
	return total, published, drafts
}

// itemSorters are the less functions listquery.Apply sorts items with, one
// per key in itemsQuery.AllowedSorts. Name, type and status compare
// case-insensitively; updated compares the time.
var itemSorters = map[string]func(a, b Item) bool{
	"name":    func(a, b Item) bool { return strings.ToLower(a.Name) < strings.ToLower(b.Name) },
	"type":    func(a, b Item) bool { return a.Type < b.Type },
	"status":  func(a, b Item) bool { return a.Status < b.Status },
	"updated": func(a, b Item) bool { return a.Updated.Before(b.Updated) },
}

// itemMatch is the filter for listquery.Apply: the item's Type must be
// one of the "type" filter values (when any), its Status one of the
// "status" values (when any), and its Name must contain the search text
// case-insensitively (when any).
//
// @pure
func itemMatch(it Item, s listquery.Spec) bool {
	if vals := s.Filters["type"]; len(vals) > 0 && !containsFold(vals, it.Type) {
		return false
	}
	if vals := s.Filters["status"]; len(vals) > 0 && !containsFold(vals, it.Status) {
		return false
	}
	if q := strings.TrimSpace(s.Search); q != "" && !strings.Contains(strings.ToLower(it.Name), strings.ToLower(q)) {
		return false
	}
	return true
}

// containsFold reports whether vals holds v, compared case-insensitively.
//
// @pure
func containsFold(vals []string, v string) bool {
	for _, x := range vals {
		if strings.EqualFold(x, v) {
			return true
		}
	}
	return false
}

// relativeTime renders how long before now t was, as the sketch's short
// labels: "now" under a minute, then "5m", "3h", "2d", "1w" (7–29 days),
// "1mo" (30–364 days), "1y". A t after now reads "now".
//
// @pure
func relativeTime(t, now time.Time) string {
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return strconv.Itoa(int(d/time.Minute)) + "m"
	case d < day:
		return strconv.Itoa(int(d/time.Hour)) + "h"
	case d < week:
		return strconv.Itoa(int(d/day)) + "d"
	case d < month:
		return strconv.Itoa(int(d/week)) + "w"
	case d < 365*day:
		return strconv.Itoa(int(d/month)) + "mo"
	default:
		return strconv.Itoa(int(d/(365*day))) + "y"
	}
}
