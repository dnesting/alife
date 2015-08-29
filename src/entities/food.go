package entities

import "fmt"

import "world"

type Food struct {
	Battery
}

func (f *Food) String() string {
	return fmt.Sprintf("[food %d]", f.Energy())
}

func NewFood(amt int) *Food {
	f := &Food{}
	f.AddEnergy(amt)
	return f
}

func (f *Food) Rune() rune {
	return '.'
}

func (f *Food) Consume(w world.World, x, y, amt int) int {
	act, e := f.AddEnergy(-amt)
	if e == 0 {
		w.Remove(x, y)
	}
	return -act
}
