// Package sim encapsulates most aspects of running a simulation.
package sim

import "sync"
import "time"

import "github.com/dnesting/alife/goalife/entities/census"
import "github.com/dnesting/alife/goalife/world"

// Sim encapsulates a world, a census and performs the Run operation on the
// runnable entities within the world.
type Sim struct {
	World  world.World
	Census *census.DirCensus

	// MutateOnDivideProb is the probability that Mutate() will be invoked on a Mutable
	// that is the product of a Divide() operation.
	MutateOnDivideProb float32

	// BodyEnergy is the amount of energy in the corpse/food component of an organism. When
	// an organism is spawned, this much energy is needed up front, and when an organism
	// dies, it is replaced with a Food pellet with this much energy.
	BodyEnergy int

	// SenseDistance is how many cells we examine to compute the amount of energy "sensed"
	// in a particular direction.
	SenseDistance int

	mu      sync.RWMutex
	wg      sync.WaitGroup
	running bool
}

// NewSim creates a new Sim with the given world.
func NewSim(w world.World) *Sim {
	return &Sim{
		World: w,
	}
}

// StopAll sets a flag that will result in subsequent invocations of
// IsStopped to return true.
func (s *Sim) StopAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.running = false
}

// IsStopped returns true if StopAll was invoked.
func (s *Sim) IsStopped() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return !s.running
}

// Runnable represents a world occupant that is capable of doing something.
type Runnable interface {
	Run(s *Sim)
}

// Time returns some int64 value representing the progress of time.  This could
// be associated with a clock, or might just be an incrementing counter.
func (s *Sim) Time() int64 {
	return time.Now().UnixNano()
}

// Start begins executing the given Runnable, updating the Census as needed.
func (s *Sim) Start(st Runnable) {
	s.wg.Add(1)
	if g, ok := st.(census.Genomer); ok {
		s.Census.Add(s.Time(), g.Genome())
	}
	go func() {
		defer s.wg.Done()
		if g, ok := st.(census.Genomer); ok {
			defer func() {
				s.Census.Remove(s.Time(), g.Genome())
			}()
		}
		st.Run(s)
	}()
}

// Run begins executing all Runnable items within the world.  This function
// returns when there are no Runnable items executing anymore.  This implies
// that the world must be seeded with one or more Runnable items before Run
// will be effective.
func (s *Sim) Run() {
	func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.running = true
	}()
	s.World.Each(func(x, y int, o world.Occupant) {
		if st, ok := o.(Runnable); ok {
			s.Start(st)
		}
	})
	s.wg.Wait()
}
