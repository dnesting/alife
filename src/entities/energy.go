package entities

import "fmt"

import "world"

type Energetic interface {
	Energy() int
	AddEnergy(amt int) (int, int)
	Consume(w world.World, x, y, amt int) int
}

type Battery struct {
	V int
}

func (e *Battery) String() string {
	return fmt.Sprintf("[battery %d]", e.V)
}

func (e *Battery) Energy() int {
	return e.V
}

func (e *Battery) AddEnergy(amt int) (int, int) {
	v := e.V + amt
	nv := v
	if nv < 0 {
		nv = 0
		amt -= v
	}
	e.V = nv
	return amt, e.V
}

func (e *Battery) Consume(w world.World, x, y, amt int) int {
	act, _ := e.AddEnergy(-amt)
	return -act
}
