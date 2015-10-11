package energy

import "fmt"
import "sync"

// Energetic describes types that have some notion of stored energy.
// The energy level should never drop below zero.
type Energetic interface {
	Energy() int
	AddEnergy(amt int) (int, int)
}

type nullEnergy struct{}

func (_ nullEnergy) Energy() int {
	return 0
}

func (_ nullEnergy) AddEnergy(_ int) (int, int) {
	return 0, 0
}

var Null = nullEnergy{}

// Battery is a simple implementation of Energetic that just stores a
// count of available energy. Its value must never be set below zero.
type Battery struct {
	mu sync.RWMutex
	V  int
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

func (e *Battery) Reset(amt int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.V = amt
}

// AddEnergy adds the given amt to the battery. amt may be negative
// to reduce the amount of energy in the battery. The amount of energy
// will never drop below zero.  Returns the actual amount of adjustment,
// and the new energy level.
func (e *Battery) AddEnergy(amt int) (adj int, newLevel int) {
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
