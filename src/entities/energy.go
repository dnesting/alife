package entities

import "world"

type Energetic interface {
	Energy() int
	AddEnergy(amt int) (int, int)
	Consume(w world.World, x, y, amt int) int
}

type energy struct {
	v int
}

func Energy(v int) Energetic {
	if v < 0 {
		v = 0
	}
	return &energy{v}
}

func (e *energy) Energy() int {
	return e.v
}

func (e *energy) AddEnergy(amt int) (int, int) {
	v := e.v + amt
	nv := v
	if nv < 0 {
		nv = 0
		amt -= v
	}
	e.v = nv
	return amt, e.v
}

func (e *energy) Consume(w world.World, x, y, amt int) int {
	act, _ := e.AddEnergy(-amt)
	return act
}
