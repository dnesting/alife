package org

import "bytes"
import "fmt"

import "entities"
import "math/rand"
import "sim"

const MutateOnDivideProb = 0.01
const BodyEnergy = 1000
const SenseDistance = 10

type SenseFilter func(o interface{}) float64

type Mutable interface {
	Mutate()
}

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

func (o *BaseOrganism) PlaceRandomly(s *sim.Sim, n Organism) {
	o.X, o.Y = s.World.PlaceRandomly(n)
}

func (o *BaseOrganism) Forward(s *sim.Sim) {
	x, y := resolveDir(o.X, o.Y, o.Dir, 1)
	if s.World.MoveIfEmpty(o.X, o.Y, x, y) == nil {
		o.X = x
		o.Y = y
	}
}

func (o *BaseOrganism) Right() {
	o.Dir = (o.Dir + 1) % 7
}

func (o *BaseOrganism) Left() {
	if o.Dir == 0 {
		o.Dir = 7
	} else {
		o.Dir--
	}
}

func (o *BaseOrganism) Neighbor(s *sim.Sim) interface{} {
	x, y := resolveDir(o.X, o.Y, o.Dir, 1)
	return s.World.At(x, y)
}

func (o *BaseOrganism) SetDir(dir int) {
	o.Dir = dir
}

func (o *BaseOrganism) Divide(s *sim.Sim, frac float32, no Organism, nb *BaseOrganism) {
	nb.Dir = rand.Intn(8)

	if m, ok := no.(Mutable); ok {
		if rand.Float32() < MutateOnDivideProb {
			m.Mutate()
		}
	}

	x, y := resolveDir(o.X, o.Y, o.Dir, 1)
	if s.World.PutIfEmpty(x, y, no) == nil {
		amt, e := o.AddEnergy(-BodyEnergy)
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

func (o *BaseOrganism) Sense(s *sim.Sim, filterFn SenseFilter) float64 {
	result := 0.0
	for dist := 1; dist <= SenseDistance; dist++ {
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

func (o *BaseOrganism) Die(s *sim.Sim, n Organism, reason string) {
	s.World.Put(o.X, o.Y, entities.NewFood(o.Energy()+BodyEnergy))
	//fmt.Printf("%v (%v) dying: %s\n", o, x, reason)
}

func (o *BaseOrganism) EatNeighbor(s *sim.Sim, amt int) {
	x, y := resolveDir(o.X, o.Y, o.Dir, 1)
	if n := s.World.At(x, y); n != nil {
		if e, ok := n.(entities.Energetic); ok {
			act := e.Consume(s.World, x, y, amt)
			o.AddEnergy(act)
		}
	}
}
