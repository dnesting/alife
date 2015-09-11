package entities

import "fmt"
import "sync"

import "github.com/dnesting/alife/goalife/world"

// Energetic describes types that have some notion of stored energy.
// The energy level should never drop below zero.
type Energetic interface {
	Energy() int
	AddEnergy(amt int) (int, int)
	Consume(w *world.World, x, y, amt int) int
}

// Battery is a simple implementation of Energetic that just stores a
// count of available energy. Its value must never be set below zero.
type Battery struct {
	V  int
	mu sync.RWMutex
}

func (e *Battery) String() string {
	return fmt.Sprintf("[battery %d]", e.V)
}

// Energy returns the current amount of energy.
func (e *Battery) Energy() int {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.V
}

// AddEnergy adds the given amt to the battery. amt may be negative
// to reduce the amount of energy in the battery. The amount of energy
// will never drop below zero.  Returns the actual amount of adjustment,
// and the new energy level.
func (e *Battery) AddEnergy(amt int) (int, int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	v := e.V + amt
	nv := v
	if nv < 0 {
		nv = 0
		amt -= v
	}
	e.V = nv
	return amt, e.V
}

// Consume works like AddEnergy with a negative value, but also permits
// implementations to do something special in the world in response to
// a change in energy levels (such as replacing an organism with a food
// pellet when it reaches zero energy).  Prefer this to AddEnergy(-amt)
// if you'd like the callee to react to the change.
func (e *Battery) Consume(w *world.World, x, y, amt int) int {
	act, _ := e.AddEnergy(-amt)
	return -act
}
