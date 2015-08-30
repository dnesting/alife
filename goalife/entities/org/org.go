// Package org defines an organism, with the ability to interact with the world.
package org

import "bytes"
import "fmt"
import "math/rand"

import "github.com/dnesting/alife/goalife/entities"
import "github.com/dnesting/alife/goalife/sim"

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

	SetDir(dir int)
	Right()
	Left()

	Neighbor(s *sim.Sim) interface{}
	Divide(s *sim.Sim, frac float32, n Organism, b *BaseOrganism)
	Sense(s *sim.Sim, fn SenseFilter) float64
	Die(s *sim.Sim, n Organism, reason string)
	EatNeighbor(s *sim.Sim, amt int)
}

// BaseOrganism encompasses the non-functional elements of an organism, including its
// energy level, orientation and position in the world.
type BaseOrganism struct {
	entities.Battery
	Dir  int
	X, Y int
}

func (o *BaseOrganism) String() string {
	var b bytes.Buffer
	b.WriteString(fmt.Sprintf("[baseorg (%d,%d) ", o.X, o.Y))
	b.WriteString(fmt.Sprintf("energy=%d", o.Energy()))
	b.WriteString(fmt.Sprintf(" dir=%d", o.Dir))
	b.WriteString("]")
	return b.String()
}

// resolveDir returns the (x, y) coordinates starting from the given (x, y) coordinates
// translated in the given direction the given number of spaces.  The returned coordinates
// do not wrap and may be negative.
func resolveDir(x, y, dir, dist int) (int, int) {
	switch dir {
	case 0:
		return x, y - dist
	case 1:
		return x + dist, y - dist
	case 2:
		return x + dist, y
	case 3:
		return x + dist, y + dist
	case 4:
		return x, y + dist
	case 5:
		return x - dist, y + dist
	case 6:
		return x - dist, y
	case 7:
		return x - dist, y - dist
	default:
		panic(fmt.Sprintf("resolveDir with out-of-range dir=%d", dir))
	}
}

// PlaceRandomly places the given organism at a random location in the world.
// The given Organism (n) must embed the receiver.
func (o *BaseOrganism) PlaceRandomly(s *sim.Sim, n Organism) {
	o.X, o.Y = s.World.PlaceRandomly(n)
}

// Forward moves the organism forward one spot if it is unoccupied. If it is
// occupied, no effect occurs.
func (o *BaseOrganism) Forward(s *sim.Sim) {
	x, y := resolveDir(o.X, o.Y, o.Dir, 1)
	if s.World.MoveIfEmpty(o.X, o.Y, x, y) == nil {
		o.X = x
		o.Y = y
	}
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
	x, y := resolveDir(o.X, o.Y, o.Dir, 1)
	return s.World.At(x, y)
}

// SetDir explicitly sets the direction of the organism to a given value.
// Organisms should generally think in relative terms and use Right and Left instead.
func (o *BaseOrganism) SetDir(dir int) {
	o.Dir = dir
}

// Divide spawns the provided new organism at the given location.
// The energy of the original and new organisms will be split according to frac (1.0 = all available energy is given to the child).
// no refers to the child organism, while nb refers to the embedded BaseOrganism within it.
func (o *BaseOrganism) Divide(s *sim.Sim, frac float32, no Organism, nb *BaseOrganism) {
	nb.Dir = rand.Intn(8)

	if m, ok := no.(Mutable); ok {
		if rand.Float32() < s.MutateOnDivideProb {
			m.Mutate()
		}
	}

	x, y := resolveDir(o.X, o.Y, o.Dir, 1)
	if s.World.PutIfEmpty(x, y, no) == nil {
		amt, e := o.AddEnergy(-s.BodyEnergy)
		if e == 0 {
			s.World.Put(x, y, entities.NewFood(-amt))
		} else {
			amt = -int((1.0 - frac) * float32(o.Energy()))
			amt, _ = o.AddEnergy(amt)
			no.AddEnergy(-amt)
			nb.X = x
			nb.Y = y
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
	result := 0.0
	for dist := 1; dist <= s.SenseDistance; dist++ {
		x, y := resolveDir(o.X, o.Y, o.Dir, dist)
		if occ := s.World.At(x, y); occ != nil {
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
	s.World.Put(o.X, o.Y, entities.NewFood(o.Energy()+s.BodyEnergy))
	//fmt.Printf("%v (%v) dying: %s\n", o, x, reason)
}

// EatNeighbor consumes at most amt of energy from the neighbor directly ahead.
// If the neighbor does not implement Energetic, no effect occurs.
func (o *BaseOrganism) EatNeighbor(s *sim.Sim, amt int) {
	x, y := resolveDir(o.X, o.Y, o.Dir, 1)
	if n := s.World.At(x, y); n != nil {
		if e, ok := n.(entities.Energetic); ok {
			act := e.Consume(s.World, x, y, amt)
			o.AddEnergy(act)
		}
	}
}
