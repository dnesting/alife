package org

import "errors"
import "fmt"
import "math"
import "math/rand"
import "sync"
import "runtime"

import "github.com/dnesting/alife/goalife/energy"
import "github.com/dnesting/alife/goalife/world/grid2d"
import "github.com/dnesting/alife/goalife/log"

const BodyEnergy = 100
const SenseFalloffExp = 2
const SenseDistance = 10

var Logger = log.Null()

type Organism struct {
	energy.Battery
	loc    grid2d.Locator
	Driver interface{}

	sync.Mutex
	Dir int
}

func (o *Organism) String() string {
	return fmt.Sprintf("[org %v e=%v d=%v %v]", o.loc, o.Energy(), o.Dir, o.Driver)
}

func (o *Organism) UseLocator(loc grid2d.Locator) {
	o.loc = loc
}

func (o *Organism) Left() {
	Logger.Printf("%v.Left()\n", o)
	o.Lock()

	o.Dir -= 1
	if o.Dir < 0 {
		o.Dir = 7
	}

	o.Unlock()
	runtime.Gosched()
}

func (o *Organism) Right() {
	Logger.Printf("%v.Right()\n", o)
	o.Lock()
	o.Dir = (o.Dir + 1) % 8
	o.Unlock()
	runtime.Gosched()
}

var ErrNoEnergy = errors.New("out of energy")

func (o *Organism) Discharge(amt int) error {
	act, _ := o.AddEnergy(-amt)
	if amt != -act {
		return ErrNoEnergy
	}
	return nil
}

func (o *Organism) Die() {
	Logger.Printf("%v.Die()\n", o)
	o.loc.Replace(energy.NewFood(o.Energy() + BodyEnergy))
	runtime.Gosched()
}

func (o *Organism) delta(dist int) (int, int) {
	switch o.Dir {
	case 0:
		return dist * 1, 0
	case 1:
		return dist * 1, dist * -1
	case 2:
		return 0, dist * -1
	case 3:
		return dist * -1, dist * -1
	case 4:
		return dist * -1, 0
	case 5:
		return dist * -1, dist * 1
	case 6:
		return 0, dist * 1
	case 7:
		return dist * 1, dist * 1
	default:
		panic(fmt.Sprintf("out of range direction %d"))
	}
}

var ErrNotEmpty = errors.New("cell occupied")

func (o *Organism) Forward() error {
	Logger.Printf("%v.Forward()\n", o)
	if err := o.Discharge(1); err != nil {
		Logger.Printf("%v.Forward: %v\n", o, err)
		return err
	}
	dx, dy := o.delta(1)
	if _, ok := o.loc.Move(dx, dy, grid2d.PutWhenNil); ok {
		runtime.Gosched()
		return nil
	}
	return ErrNotEmpty
}

func Random() *Organism {
	return &Organism{Dir: rand.Intn(8)}
}

var PutWhenFood = func(orig, n interface{}) bool {
	if orig == nil {
		return true
	}
	if _, ok := orig.(*energy.Food); ok {
		return true
	}
	return false
}

func (o *Organism) Divide(driver interface{}, energyFrac float64) (*Organism, error) {
	Logger.Printf("%v.Divide(%v, %v)\n", o, driver, energyFrac)
	if err := o.Discharge(5); err != nil {
		return nil, err
	}

	n := Random()
	n.Driver = driver
	dx, dy := o.delta(1)
	if _, loc := o.loc.Put(dx, dy, n, PutWhenFood); loc != nil {
		energy.Transfer(n, o, int(float64(o.Energy())*energyFrac))
		runtime.Gosched()
		return n, nil
	}
	return nil, ErrNotEmpty
}

func (o *Organism) Sense(fn func(o interface{}) float64) float64 {
	Logger.Printf("%v.Sense(%v)\n", o, fn)
	var e float64
	if fn == nil {
		fn = func(_ interface{}) float64 { return 1.0 }
	}
	for i := 1; i <= SenseDistance; i++ {
		if n := o.loc.Get(o.delta(i)); n != nil {
			if n, ok := n.(energy.Energetic); ok {
				e += float64(n.Energy()) * fn(n) / math.Pow(float64(i), SenseFalloffExp)
			}
		}
	}
	runtime.Gosched()
	return e
}

func (o *Organism) Eat(amt int) (int, error) {
	Logger.Printf("%v.Eat(%v)\n", o, amt)
	if err := o.Discharge(amt / 100); err != nil {
		return 0, err
	}
	if n := o.loc.Get(o.delta(1)); n != nil {
		Logger.Printf("- got %v\n", n.Value())
		if n, ok := n.Value().(energy.Energetic); ok {
			Logger.Printf("- is energetic\n")
			amt, _, _ = energy.Transfer(o, n, amt)
			Logger.Printf("- transferred %v\n", amt)
			runtime.Gosched()
			return -amt, nil
		} else {
			Logger.Printf("- not energetic\n")
		}
	} else {
		Logger.Printf("- empty\n")
	}
	return 0, nil
}
