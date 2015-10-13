package energy

import "fmt"
import "sync/atomic"

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
	V int32
}

func (e *Battery) String() string {
	return fmt.Sprintf("[battery %d]", e.V)
}

// Energy returns the current amount of energy.
func (e *Battery) Energy() int {
	return int(atomic.LoadInt32(&e.V))
}

func (e *Battery) Reset(amt int) {
	atomic.StoreInt32(&e.V, int32(amt))
}

// AddEnergy adds the given amt to the battery. amt may be negative
// to reduce the amount of energy in the battery. The amount of energy
// will never drop below zero.  Returns the actual amount of adjustment,
// and the new energy level.
func (e *Battery) AddEnergy(amt int) (adj int, newLevel int) {
	for {
		orig := atomic.LoadInt32(&e.V)
		v := orig + int32(amt)
		nv := v
		if nv < 0 {
			nv = 0
			amt -= int(v)
		}
		if atomic.CompareAndSwapInt32(&e.V, orig, nv) {
			return amt, int(nv)
		}
	}
}

// Transfer moves at most amt units of energy from src to dest. Neither
// entity's energy will drop below zero. Returns the actual amount
// transferred, and the remaining energy in dest and src.
func Transfer(dest, src Energetic, amt int) (actual int, destE int, srcE int) {
	if amt < 0 {
		dest, src = src, dest
		amt = -amt
	}
	actual, srcE = src.AddEnergy(-amt)
	_, destE = dest.AddEnergy(-actual)
	return -actual, destE, srcE
}
