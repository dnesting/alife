package entities

import "fmt"

import "github.com/dnesting/alife/goalife/world"

// Food is a type of battery that, when its energy drops to zero, is removed from the world.
type Food struct {
	Battery
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

// Rune returns a fixed '.' rune to render a food pellet in the world.
func (f *Food) Rune() rune {
	e := f.Energy()
	switch {
	case e > 4000:
		return '⁙'
	case e > 3000:
		return '⁘'
	case e > 2000:
		return '⁖'
	case e > 1000:
		return '⁚'
	default:
		return '·'
	}
}

// Consume consumes the given amt of energy, removing the food pellet from
// the world once its energy level drops to zero.
func (f *Food) Consume(loc world.Locator, amt int) int {
	act, e := f.AddEnergy(-amt)
	if e == 0 {
		loc.Remove()
	}
	return -act
}
