// Package sim encapsulates most aspects of running a simulation.
package sim

import "fmt"
import "io"
import "sync"
import "time"

import "github.com/dnesting/alife/goalife/entities/census"
import "github.com/dnesting/alife/goalife/world"

// Sim encapsulates a world, a census and performs the Run operation on the
// runnable entities within the world.
type Sim struct {
	World  *world.World
	Census census.Census

	// MutateOnDivideProb is the probability that Mutate() will be invoked on a Mutable
	// that is the product of a Divide() operation.
	MutateOnDivideProb float32

	// BodyEnergy is the amount of energy in the corpse/food component of an organism. When
	// an organism is spawned, this much energy is needed up front, and when an organism
	// dies, it is replaced with a Food pellet with this much energy.
	BodyEnergy int

	// InitialEnergy is the amount of energy initially granted to a new organism.
	InitialEnergy int

	// SenseDistance is how many cells we examine to compute the amount of energy "sensed"
	// in a particular direction.
	SenseDistance int

	// MinimumOrgs is the number of organisms we will maintain in the world.  If the number
	// falls below this value, we will add more using OrgFactory.
	MinimumOrgs int

	// FractionFromHistory is the ratio of organisms added when trying to meet MinimumOrgs
	// that are taken from Census history rather than randomly-generated.
	FractionFromHistory float32

	// OrgFactory creates new organisms used to seed the world (whenever the count of items
	// falls below MinimumOrgs).  If nil, and FractionFromHistory is non-zero, populates
	// entirely from history.  Otherwise, MinimumOrgs has no effect.
	OrgFactory func() interface{}

	// Tracer is an optional io.Writer where tracing messages will be written.
	Tracer io.Writer

	mu      sync.RWMutex
	pending []Runnable
	wg      sync.WaitGroup
	running bool
}

// NewSim creates a new Sim with the given world.
func NewSim(w *world.World, c census.Census) *Sim {
	s := &Sim{
		World:  w,
		Census: c,
	}

	s.World.EachLocation(func(x, y int, v interface{}) {
		if r, ok := v.(Runnable); ok {
			s.Start(r)
		}
	})

	c.OnChange(func(_ census.Census, _ census.Cohort, _ bool) {
		if !s.IsStopped() {
			s.ensureMinimumOrgs()
		}
	})

	return s
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
	if s.IsStopped() {
		s.pending = append(s.pending, st)
	} else {
		s.startRunning(st)
	}
}

func (s *Sim) startRunning(st Runnable) {
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

func (s *Sim) ensureMinimumOrgs() {
	if s.OrgFactory == nil {
		return
	}
	for c := s.Census.Count(); c < s.MinimumOrgs; c++ {
		i := s.OrgFactory()
		if r, ok := i.(Runnable); ok {
			s.Start(r)
		}
	}
}

// Run begins executing all Runnable items within the world.  This function
// returns when there are no Runnable items executing anymore, which may be
// never.
func (s *Sim) Run() {
	s.ensureMinimumOrgs()
	func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.running = true
		for _, r := range s.pending {
			s.startRunning(r)
		}
		s.pending = nil
	}()
	s.wg.Wait()
}

func (s *Sim) T(e interface{}, msg string, args ...interface{}) {
	if s.Tracer != nil {
		a := []interface{}{e}
		a = append(a, args...)
		fmt.Fprintf(s.Tracer, fmt.Sprintf("%%v: %s\n", msg), a...)
	}
}
