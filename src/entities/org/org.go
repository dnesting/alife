package org

import "hash/crc32"
import "fmt"
import "math/rand"

import "entities"
import "world"

type Organism struct {
	entities.Energetic

	Genome uint32

	Code []byte
	cpu  *Cpu
	dir  int
	w    world.World
	x, y int
}

func (o *Organism) String() string {
	return fmt.Sprintf("[org (%d,%d) e=%d g=%d %v]", o.x, o.y, o.Energy(), o.Genome, o.cpu)
}

var runes string = "abcdefhijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func (o *Organism) Rune() rune {
	return rune(runes[int(o.Genome)%len(runes)])
}

func Random() *Organism {
	o := New()
	o.SetCode(RandomBytecode())
	o.dir = rand.Intn(8)
	return o
}

func New() *Organism {
	return &Organism{
		Energetic: entities.Energy(0),
		cpu:       NewCpu(),
	}
}

func NewFrom(o *Organism) *Organism {
	return &Organism{
		Energetic: entities.Energy(0),
		cpu:       NewCpuFrom(o.cpu),
		w:         o.w,
		x:         o.x,
		y:         o.y,
	}
}

func (o *Organism) SetCode(data []byte) (orig []byte) {
	orig = o.Code
	o.Code = data
	o.cpu.SetCode(data)
	o.Genome = crc32.ChecksumIEEE(data)
	return
}

func (o *Organism) Step(w world.World, x, y int) {
	o.w = w
	o.x = x
	o.y = y
	if err := o.cpu.Step(o); err != nil {
		o.Die(err.Error())
	}
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

func (o *Organism) Forward() error {
	x, y := resolveDir(o.x, o.y, o.dir, 1)
	o.w.MoveIfEmpty(o.x, o.y, x, y)
	return nil
}

func (o *Organism) Right() {
	o.dir = (o.dir + 1) % 7
}

func (o *Organism) Left() {
	if o.dir == 0 {
		o.dir = 7
	} else {
		o.dir--
	}
}

func (o *Organism) Neighbor() (interface{}, int, int) {
	x, y := resolveDir(o.x, o.y, o.dir, 1)
	return o.w.At(x, y), x, y
}

func (o *Organism) Divide(frac float32) {
	frac = 0.5
	n := NewFrom(o)
	n.SetCode(o.cpu.Mutated())
	n.dir = rand.Intn(8)
	x, y := resolveDir(o.x, o.y, o.dir, 1)
	if o.w.PutIfEmpty(x, y, n) == nil {
		amt, e := o.AddEnergy(-BodyEnergy)
		if e == 0 {
			o.w.Put(x, y, entities.NewFood(-amt))
		} else {
			amt = -int((1.0 - frac) * float32(o.Energy()))
			amt, _ = o.AddEnergy(amt)
			n.AddEnergy(-amt)
			n.x = x
			n.y = y
		}
	}
	return
}

const SenseDistance = 10

func (o *Organism) Sense(ignoreSameGenome bool) int {
	result := 0
	for dist := 1; dist <= SenseDistance; dist++ {
		x, y := resolveDir(o.x, o.y, o.dir, dist)
		if occ := o.w.At(x, y); occ != nil {
			erg := 0
			if e, ok := occ.(Organism); ok && ignoreSameGenome {
				if e.Genome != o.Genome {
					erg = e.Energy()
				}
			} else if e, ok := occ.(entities.Energetic); ok {
				erg = e.Energy()
			}
			result += int(float32(erg) * (1.0 / float32(dist)))
		}
	}
	return result
}

const BodyEnergy = 500

func (o *Organism) Die(reason string) {
	o.w.ReplaceIfEqual(o.x, o.y, o, entities.NewFood(o.Energy()+BodyEnergy))
}
