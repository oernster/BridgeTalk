package services

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/selection"
)

// ErrNoSuchMoment is returned for a switch asked of a moment or a category Chatter does not list.
var ErrNoSuchMoment = errors.New("no such moment")

// Moment is one moment Chatter lists: its cue and whether it is switched on.
type Moment struct {
	Cue cue.Cue
	On  bool
}

// Category is one category Chatter lists, holding its moments in the table's order.
type Category struct {
	Name    string
	Moments []Moment
}

// ChatterService holds which moments are switched off and keeps them for the next run (FR-621 to
// FR-633).
//
// It is built once, at start, rather than with each cast, so casting another voice leaves every
// switch as it stands (FR-630). The switches are a value swapped in whole: the poll loop reads them
// on its goroutine while the window changes them on its own, so a reader always holds one whole set.
// Changes are taken one at a time, so two presses arriving together cannot lose either.
type ChatterService struct {
	table    cue.Table
	store    ports.SettingsStore
	switches atomic.Pointer[selection.Switches]
	changing sync.Mutex
}

// NewChatterService starts with the switches the store kept (FR-629). With no store or a store
// keeping none, every moment is on (FR-628, FR-632).
func NewChatterService(table cue.Table, store ports.SettingsStore) *ChatterService {
	var kept []cue.ID
	if store != nil {
		for _, id := range store.Load().SwitchedOff {
			kept = append(kept, cue.ID(id))
		}
	}
	service := &ChatterService{table: table, store: store}
	switches := selection.NewSwitches(kept)
	service.switches.Store(&switches)
	return service
}

// Off reports whether a moment is switched off; it is the Switchboard the engine asks (FR-622).
func (c *ChatterService) Off(id cue.ID) bool { return c.switches.Load().Off(id) }

// Categories answers every category in the table's order, each with its moments and whether each is
// switched on (FR-727).
func (c *ChatterService) Categories() []Category {
	switches := c.switches.Load()
	names := c.table.Categories()
	categories := make([]Category, len(names))
	place := make(map[string]int, len(names))
	for at, name := range names {
		categories[at].Name = name
		place[name] = at
	}
	for _, item := range c.table.All() {
		at, listed := place[item.Category()]
		if !listed || !item.Switchable() {
			continue
		}
		categories[at].Moments = append(categories[at].Moments, Moment{Cue: item, On: !switches.Off(item.ID())})
	}
	return categories
}

// SetMoment switches one moment on or off (FR-729).
func (c *ChatterService) SetMoment(id cue.ID, on bool) error {
	for _, category := range c.Categories() {
		for _, listed := range category.Moments {
			if listed.Cue.ID() == id {
				return c.set(on, id)
			}
		}
	}
	return fmt.Errorf("%w: Chatter lists no moment %s", ErrNoSuchMoment, id)
}

// SetCategory switches every moment in one category on or off (FR-731).
func (c *ChatterService) SetCategory(name string, on bool) error {
	for _, category := range c.Categories() {
		if category.Name == name {
			return c.set(on, idsOf(category.Moments)...)
		}
	}
	return fmt.Errorf("%w: Chatter lists no category %q", ErrNoSuchMoment, name)
}

// SetAll switches every moment on or off (FR-732).
func (c *ChatterService) SetAll(on bool) error {
	var ids []cue.ID
	for _, category := range c.Categories() {
		ids = append(ids, idsOf(category.Moments)...)
	}
	return c.set(on, ids...)
}

// set applies a change at once, then keeps it beside every other setting (FR-627, FR-629). A change
// that cannot be kept still applies until the application closes; the error says why it was not kept
// (FR-633). What is kept names only moments the table holds, so an id no cue has is let go (FR-631).
func (c *ChatterService) set(on bool, ids ...cue.ID) error {
	c.changing.Lock()
	defer c.changing.Unlock()

	next := c.switches.Load().With(on, ids...)
	c.switches.Store(&next)
	if c.store == nil {
		return nil
	}
	held := c.store.Load()
	kept := next.Kept(c.table)
	held.SwitchedOff = make([]string, 0, len(kept))
	for _, id := range kept {
		held.SwitchedOff = append(held.SwitchedOff, string(id))
	}
	if err := c.store.Save(held); err != nil {
		return fmt.Errorf("keeping the switches: %w", err)
	}
	return nil
}

// idsOf answers the ids of the moments given.
func idsOf(moments []Moment) []cue.ID {
	ids := make([]cue.ID, 0, len(moments))
	for _, each := range moments {
		ids = append(ids, each.Cue.ID())
	}
	return ids
}
