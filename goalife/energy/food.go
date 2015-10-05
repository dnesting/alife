package energy

import "fmt"

import "github.com/dnesting/alife/goalife/world/grid2d"

// Food is a type of battery that, when its energy drops to zero, its OnEmpty func is called.
type Food struct {
	Battery
	loc grid2d.Locator
}

func (f *Food) String() string {
	return fmt.Sprintf("[food %d]", f.Energy())
}

// NewFood creates a new Food instance with the given energy level.
func NewFood(amt int) *Food {
	f := &Food{}
	f.AddEnergy(amt)
	return f
}

func (f *Food) AddEnergy(amt int) (adj int, newLevel int) {
	adj, newLevel = f.Battery.AddEnergy(amt)
	if adj != 0 && newLevel == 0 && f.loc != nil {
		f.loc.Remove()
	}
	return adj, newLevel
}

func (f *Food) UseLocator(l grid2d.Locator) {
	f.loc = l
}
