package food

import "fmt"
import "sync"

import "github.com/dnesting/alife/goalife/energy"
import "github.com/dnesting/alife/goalife/grid2d"

// Food is a type of battery that, when its energy drops to zero, its OnEmpty func is called.
type Food struct {
	energy.Battery

	mu  sync.Mutex
	loc grid2d.Locator
}

var foodPool = sync.Pool{New: func() interface{} { return &Food{} }}

func (f *Food) String() string {
	return fmt.Sprintf("[food %d]", f.Energy())
}

// NewFood creates a new Food instance with the given energy level.
func New(amt int) *Food {
	f := foodPool.Get().(*Food)
	f.Reset(amt)
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

func (f *Food) AddEnergy(amt int) (adj int, newLevel int) {
	adj, newLevel = f.Battery.AddEnergy(amt)
	if adj != 0 && newLevel == 0 {
		f.invalidate()
		foodPool.Put(f)
	}
	return adj, newLevel
}

func (f *Food) UseLocator(l grid2d.Locator) {
	f.loc = l
}
