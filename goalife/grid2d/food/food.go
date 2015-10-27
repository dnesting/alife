// Package food defines a simple energy store with awareness of its
// existence in a grid2d.
package food

import "fmt"
import "sync"

import "github.com/dnesting/alife/goalife/energy"
import "github.com/dnesting/alife/goalife/grid2d"

// Food is a type of energy store that, when its energy drops to
// zero, calls loc.RemoveWithPlaceholder(energy.Null).
type Food struct {
	energy.Store

	mu  sync.Mutex
	loc grid2d.Locator
}

// Try to re-use Food instances because these are little objects that are
// frequently instantiated and destroyed.
var foodPool = sync.Pool{New: func() interface{} { return &Food{} }}

func (f *Food) String() string {
	return fmt.Sprintf("[food %d]", f.Energy())
}

// New creates a new Food instance with the given energy level.  Attempts to
// allocate these from a sync.Pool if any are available.
func New(amt int) *Food {
	f := foodPool.Get().(*Food)
	f.ResetEnergy(amt)
	return f
}

func (f *Food) invalidate() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.loc != nil {
		f.loc.RemoveWithPlaceholder(energy.Null)
		f.loc = nil
	}
}

// AddEnergy adds amt to the food's energy store.  If this causes
// the energy level to drop to zero and a locator was provided
// with UseLocator, it will be used to remove the food.
func (f *Food) AddEnergy(amt int) (adj int, newLevel int) {
	adj, newLevel = f.Store.AddEnergy(amt)
	if adj != 0 && newLevel == 0 {
		f.invalidate()
		foodPool.Put(f)
	}
	return adj, newLevel
}

// UseLocator associates this food instance with a grid2d.Locator,
// which will be used to remove the food instance when its energy
// level drops to zero.
func (f *Food) UseLocator(l grid2d.Locator) {
	f.loc = l
}
