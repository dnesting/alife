// Package org defines an organism, with the ability to interact with the world.
package org

import "bytes"
import "fmt"
import "math/rand"

import "github.com/dnesting/alife/goalife/entities"
import "github.com/dnesting/alife/goalife/sim"
import "github.com/dnesting/alife/goalife/world"

// SenseFilter is used to determine if an item that occupies a cell during a Sense operation
// should contribute to the aggregated energy value for that operation.
type SenseFilter func(o interface{}) float64

// Mutable is something that can be mutated when the need arises.  When Mutate() is invoked,
// the Mutable instance should modify itself however is appropriate.
type Mutable interface {
	Mutate()
}

// Organism is the interface used by driving implementations to interact with the world
// around it.
type Organism interface {
	entities.Energetic

	Forward(s *sim.Sim)
	Right()
	Left()

	Neighbor(s *sim.Sim) interface{}
	Divide(s *sim.Sim, frac float32, n Organism, b *BaseOrganism)
	Sense(s *sim.Sim, fn SenseFilter) float64
	Die(s *sim.Sim, n Organism, reason string)
	EatNeighbor(s *sim.Sim, amt int)
}

// BaseOrganism encompasses the non-functional elements of an organism, including its
// energy level and orientation.
type BaseOrganism struct {
	entities.Battery
	Dir int
	Loc world.Locator
}

func (o *BaseOrganism) String() string {
	var b bytes.Buffer
	b.WriteString(fmt.Sprintf("[baseorg (%s)", o.Loc))
	b.WriteString(fmt.Sprintf(" e=%d", o.Energy()))
	b.WriteString(fmt.Sprintf(" dir=%d", o.Dir))
	b.WriteString("]")
	return b.String()
}

// delta returns the (dx, dy) coordinate delta from the given direction and distance.
func delta(dir, dist int) (int, int) {
	switch dir {
	case 0:
		return 0, -dist
	case 1:
		return dist, -dist
	case 2:
		return dist, 0
	case 3:
		return dist, dist
	case 4:
		return 0, dist
	case 5:
		return -dist, dist
	case 6:
		return -dist, 0
	case 7:
		return -dist, -dist
	default:
		panic(fmt.Sprintf("delta with out-of-range dir=%d", dir))
	}
}

// PlaceRandomly places the given organism at a random location in the world.
// The given Organism (n) must embed the receiver.
func (o *BaseOrganism) PlaceRandomly(s *sim.Sim, n Organism) {
	o.Loc = s.World.PlaceRandomly(n)
	s.T(n, "placed at: %v", o.Loc)
}

// Forward moves the organism forward one spot if it is unoccupied. If it is
// occupied, no effect occurs.
func (o *BaseOrganism) Forward(s *sim.Sim) {
	s.T(o, "forward")
	dx, dy := delta(o.Dir, 1)
	o.Loc.MoveIfEmpty(dx, dy)
}

// Right turns the organism one direction to the right.
func (o *BaseOrganism) Right() {
	o.Dir = (o.Dir + 1) % 7
}

// Left turns the organism one direction to the left.
func (o *BaseOrganism) Left() {
	if o.Dir == 0 {
		o.Dir = 7
	} else {
		o.Dir--
	}
}

// Neighbor returns the occupant of the cell directly ahead.
// If the cell has no occupant, returns nil.
func (o *BaseOrganism) Neighbor(s *sim.Sim) interface{} {
	dx, dy := delta(o.Dir, 1)
	s.T(o, "neighbor (%d,%d)", dx, dy)
	return o.Loc.Relative(dx, dy).Value()
}

// Divide spawns the provided new organism at the given location.
// The energy of the original and new organisms will be split according to frac (1.0 = all available energy is given to the child).
// no refers to the child organism, while nb refers to the embedded BaseOrganism within it.
func (o *BaseOrganism) Divide(s *sim.Sim, frac float32, no Organism, nb *BaseOrganism) {
	s.T(o, "divide(%.2f, %v)", frac, no)
	nb.Dir = rand.Intn(8)

	if m, ok := no.(Mutable); ok {
		if rand.Float32() < s.MutateOnDivideProb {
			m.Mutate()
		}
	}

	dx, dy := delta(o.Dir, 1)
	if l := o.Loc.PutIfEmpty(dx, dy, no); l != nil {
		amt, e := o.AddEnergy(-s.BodyEnergy)
		if e == 0 {
			l = l.Replace(entities.NewFood(-amt))
			s.T(o, "- aborting: %v", l)
		} else {
			amt = -int(frac * float32(o.Energy()))
			amt, _ = o.AddEnergy(amt)
			no.AddEnergy(-amt)
			nb.Loc = l
			s.T(o, "- child: %v: %v", l, no)
			if nr, ok := no.(sim.Runnable); ok {
				s.Start(nr)
			}
		}
	}
	return
}

// Sense returns the amount of energy seen in the area in front of the organism.
// The observed energy has exponential falloff.  An optional SenseFilter may be
// provided, which allows for an additional multiplier to be applied to each item
// observed (for instance, to ignore organisms that have the same genome).
func (o *BaseOrganism) Sense(s *sim.Sim, filterFn SenseFilter) float64 {
	s.T(o, "sense")
	result := 0.0
	for dist := 1; dist <= s.SenseDistance; dist++ {
		dx, dy := delta(o.Dir, dist)
		if occ := o.Loc.Relative(dx, dy).Value(); occ != nil {
			if e, ok := occ.(entities.Energetic); ok {
				mult := 1.0
				if filterFn != nil {
					mult = filterFn(occ)
				}
				result += float64(e.Energy()) * mult * (1.0 / float64(dist))
			}
		}
	}
	return result
}

// Die replaces the organism with an appropriate food pellet. It is required that
// the organism's goroutine exit without performing any further operations after
// this function is called.
func (o *BaseOrganism) Die(s *sim.Sim, n Organism, reason string) {
	s.T(o, "dying: %s", reason)
	o.Loc.Replace(entities.NewFood(o.Energy() + s.BodyEnergy))
}

// EatNeighbor consumes at most amt of energy from the neighbor directly ahead.
// If the neighbor does not implement Energetic, no effect occurs.
func (o *BaseOrganism) EatNeighbor(s *sim.Sim, amt int) {
	dx, dy := delta(o.Dir, 1)
	s.T(o, "eat (%d) (%d,%d)", amt, dx, dy)
	if nloc := o.Loc.Relative(dx, dy); nloc != nil {
		if e, ok := nloc.Value().(entities.Energetic); ok {
			o.AddEnergy(e.Consume(nloc, amt))
		}
	}
}
