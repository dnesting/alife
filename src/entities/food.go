package entities

import "world"

type Food struct {
	Energetic
}

func NewFood(amt int) *Food {
	return &Food{Energy(amt)}
}

func (f *Food) Rune() rune {
	return '.'
}

func (f *Food) Consume(w world.World, x, y, amt int) int {
	act, e := f.AddEnergy(-amt)
	if e == 0 {
		w.Put(x, y, nil)
	}
	return act
}
