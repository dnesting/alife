// Package org describes an "organism" that has a lifecycle, energy store,
// and interacts with the grid2d.Grid that it inhabits using more organic wrappers around
// its grid2d.Locator.
package org

import "errors"
import "fmt"
import "math"
import "math/rand"
import "sync"
import "runtime"

import "github.com/dnesting/alife/goalife/energy"
import "github.com/dnesting/alife/goalife/grid2d"
import "github.com/dnesting/alife/goalife/grid2d/food"
import "github.com/dnesting/alife/goalife/log"

// An organism's "body" is considered to have this much energy.  It costs at least this
// much energy for one organism to create another, and when an organism dies, it is replaced
// with Food storing this much energy.
const BodyEnergy = 1000

// SenseFalloffExp describes the exponential falloff as Sense range increases.
const SenseFalloffExp = 2

// SenseDistance is the maximum distance we look out to satisfy Sense calls.
const SenseDistance = 10

var Logger = log.Null()

// Organism represents an occupant of a Grid that has a more organically-inspired lifecycle,
// energy store and direction.  By itself, it doesn't do anything.  It requires additional
// functionality to "drive" it by invoking its methods to inspect and navigate its environment.
// An Organism's direction can be any of 8 values representing the four cardinal compass directions
// and one degree in between each (i.e, north, north-west, west, etc.).
//
// Most methods have an energy cost associated with them, and can return ErrNoEnergy if the
// organism's energy is exhausted.  Callers are expected to terminate execution and invoke the
// organism's Die method in this case.
type Organism struct {
	energy.Store
	loc    grid2d.Locator
	Driver interface{}

	mu  sync.Mutex
	Dir int
}

func (o *Organism) String() string {
	return fmt.Sprintf("[org %v e=%v d=%c %v]", o.loc, o.Energy(), o.Arrow(), o.Driver)
}

// UseLocator specifies the grid2d.Locator that the organism should use to inspect and
// navigate its environment.  This is normally invoked implicitly when the organism is
// placed in a Grid and should not normally be called.
func (o *Organism) UseLocator(loc grid2d.Locator) {
	o.loc = loc
}

// Left causes the organism to rotate its direction counter-clockwise once (i.e.,
// from north to north-west).
func (o *Organism) Left() {
	Logger.Printf("%v.Left()\n", o)
	o.mu.Lock()

	o.Dir -= 1
	if o.Dir < 0 {
		o.Dir = 7
	}

	o.mu.Unlock()
	runtime.Gosched()
}

// Right causes the organism to rotate its direction clockwise once (i.e.,
// from north to north-east).
func (o *Organism) Right() {
	Logger.Printf("%v.Right()\n", o)
	o.mu.Lock()
	o.Dir = (o.Dir + 1) % 8
	o.mu.Unlock()
	runtime.Gosched()
}

// ErrNoEnergy is returned from methods to signal that there is insufficient energy
// to perform the requested action.
var ErrNoEnergy = errors.New("out of energy")

// Discharge attempts to reduce the energy store of the organism by amt.  Returns
// ErrNoEnergy if this resulted in reducing the energy store to zero.
func (o *Organism) Discharge(amt int) error {
	act, _ := o.AddEnergy(-amt)
	if amt != -act {
		return ErrNoEnergy
	}
	return nil
}

// Die causes the organism to terminate its existence.  It will be replaced with
// an item of Food storing the same amount of energy as the organism plus the
// base BodyEnergy.
func (o *Organism) Die() {
	Logger.Printf("%v.Die()\n", o)
	o.loc.Replace(food.New(o.Energy() + BodyEnergy))
	runtime.Gosched()
}

// Arrow returns an arrow rune representing the direction the organism is pointing.
func (o *Organism) Arrow() rune {
	switch o.Dir {
	case 0:
		return '→'
	case 1:
		return '↗'
	case 2:
		return '↑'
	case 3:
		return '↖'
	case 4:
		return '←'
	case 5:
		return '↙'
	case 6:
		return '↓'
	case 7:
		return '↘'
	default:
		panic(fmt.Sprintf("out of range direction %d", o.Dir))
	}
}

// delta returns the relative coordinates of the cell dist cells
// away in the organisms direction.
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
		panic(fmt.Sprintf("out of range direction %d", o.Dir))
	}
}

// ErrNotEmpty is returned when an operation requires occupying a cell
// that is occupied by something that can't be replaced.
var ErrNotEmpty = errors.New("cell occupied")

// Forward attempts to move the organism forward one cell, in the
// direction the organism is pointing.  Returns ErrNoEnergy if the
// organism's energy is exhausted or ErrNotEmpty if the cell is occupied
// by something else.
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

// Random generates an organism pointing in a random direction.  The
// resulting organism has no driver and is not associated with a locator.
func Random() *Organism {
	return &Organism{Dir: rand.Intn(8)}
}

// PutWhenFood is a grid2d.PutWhenFunc that returns true if the cell is
// unoccupied, or only contains an item of Food.
var PutWhenFood = func(orig, n interface{}) bool {
	if orig == nil {
		return true
	}
	if _, ok := orig.(*food.Food); ok {
		return true
	}
	return false
}

// Divide spawns a new organism in the neighboring cell in the direction the
// organism is pointing.  Energy from the parent, multiplied by energyFrac, will
// be transferred to the child to give it something to start off with.  The
// returns organism will be associated with a grid2d.Locator and given driver,
// but still requires the caller spawn a goroutine to drive it.  Returns nil and
// an error if there was insufficient energy to divide, or if the cell the child
// would be spawned within is already occupied by anything other than Food.
func (o *Organism) Divide(driver interface{}, energyFrac float64) (*Organism, error) {
	Logger.Printf("%v.Divide(%v, %v)\n", o, driver, energyFrac)
	if err := o.Discharge(BodyEnergy); err != nil {
		return nil, err
	}

	n := Random()
	n.Driver = driver
	dx, dy := o.delta(1)
	if _, loc := o.loc.Put(dx, dy, n, PutWhenFood); loc != nil {
		energy.Transfer(n, o, int(float64(o.Energy())*energyFrac))
		Logger.Printf("- parent: %v\n", o)
		Logger.Printf("-  child: %v\n", n)
		runtime.Gosched()
		return n, nil
	}
	return nil, ErrNotEmpty
}

// Sense detects energy outward some distance in the direction the organism points.
// For each occupant found, fn will be called and should return a number from 0.0 to 1.0
// which is a multiplier that should be applied to the energy level of the occupant,
// permitting the caller to filter, attenuate or amplify the energy level of an occupant
// based on caller-determined criteria.  If nil, no multiplier will be assessed against
// occupants.  Exponential falloff will be applied on top of this, so that nearer occupants
// will contribute more to the returned energy level than more distant occupants.
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

// Eat attempts to transfer energy from the occupant in the neighboring cell in the
// direction the organism points.  Returns the amount transferred successfully or
// an error if there was insufficient energy to complete the action.
func (o *Organism) Eat(amt int) (int, error) {
	Logger.Printf("%v.Eat(%v)\n", o, amt)
	if err := o.Discharge(int(math.Ceil(float64(amt) / 100.0))); err != nil {
		return 0, err
	}
	if n := o.loc.Get(o.delta(1)); n != nil {
		Logger.Printf("- got %v\n", n.Value())
		if n, ok := n.Value().(energy.Energetic); ok {
			amt, _, _ = energy.Transfer(o, n, amt)
			Logger.Printf("- transferred %v\n", amt)
			Logger.Printf("  - %v\n", o)
			Logger.Printf("  - %v\n", n)
			runtime.Gosched()
			return -amt, nil
		} else {
			Logger.Printf("- not energetic\n")
		}
	}
	return 0, nil
}
