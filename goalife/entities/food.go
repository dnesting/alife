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
	return '.'
}

// Consume consumes the given amt of energy, removing the food pellet from
// the world once its energy level drops to zero.
func (f *Food) Consume(w world.World, x, y, amt int) int {
	act, e := f.AddEnergy(-amt)
	if e == 0 {
		w.Remove(x, y)
	}
	return -act
}
