// Package world defines the world within which organisms and other occupants will exist.
package world

// See comments in entity.go for rules around holding mutexes to avoid deadlock.

import "fmt"
import "io"
import "math/rand"
import "sync"

type Update struct {
	X, Y int
	V    *interface{}
}

// World is a place within which occupants can exist.  It contains various functions for
// retrieving and manipulating items by their (x, y) coordinates.  It is implemented as
// a toroidal 2D grid.
type World struct {
	mu       sync.RWMutex
	Grid     Grid
	EmptyFn  func(o interface{}) bool
	UpdateFn func(w *World)
	Tracer   io.Writer

	subs []chan<- []Update
}

func (w *World) Subscribe(ch chan<- []Update) {
	w.subs = append(w.subs, ch)
}

func (w *World) get(x, y int) *Entity {
	var e *Entity
	o := w.Grid.Get(x, y)
	if o != nil {
		e = o.(*Entity)
	}
	return e
}

type worldly interface {
	UseWorld(w *World)
}

func (w *World) GobEncode() ([]byte, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.Grid.GobEncode()
}

func (w *World) GobDecode(stream []byte) error {
	if err := w.Grid.GobDecode(stream); err != nil {
		return err
	}
	w.Grid.Each(func(x, y int, v interface{}) {
		if o, ok := v.(worldly); ok {
			o.UseWorld(w)
		}
	})
	return nil
}

func (w *World) Wrap(x, y int) (int, int) {
	return w.Grid.Wrap(x, y)
}

func (w *World) isEmpty(o *Entity) (ok bool) {
	defer func() { w.T(w, "isEmpty(%v) = %v", o, ok) }()
	if o == nil {
		w.T(w, "returning true because o == nil")
		return true
	}
	if w.EmptyFn == nil {
		w.T(w, "returning false because w.EmptyFn == nil")
		return false
	}
	w.T(w, "returning whatever EmptyFn returns")
	return w.EmptyFn(o.Value())
}

func (w *World) Width() int {
	return w.Grid.Width()
}

func (w *World) Height() int {
	return w.Grid.Height()
}

func (w *World) notify(u []Update) {
	if len(u) == 0 {
		return
	}
	for _, c := range w.subs {
		c <- u
	}
}

func (w *World) validateCoords(x, y int) {
	if x >= w.Width() || y >= w.Height() || x < 0 || y < 0 {
		panic(fmt.Sprintf("(%d, %d) outside of world bounds (%d, %d)", x, y, w.Width(), w.Height()))
	}
}

func (w *World) createEntity(x, y int, value interface{}) *Entity {
	w.validateCoords(x, y)
	return &Entity{
		w: w,
		X: x,
		Y: y,
		v: value,
	}
}

func (w *World) putLocked(x, y int, value interface{}, update *[]Update) (e *Entity) {
	defer func() { w.T(w, "putLocked(%d,%d, %v) = %v", x, y, value, e) }()
	e = w.createEntity(x, y, value)
	w.Grid.Put(x, y, e)
	*update = append(*update, Update{x, y, &e.v})
	return e
}

// at returns the occupant at the given (x, y) coordinate.  Concurrent
// execution may mean that the occupant has moved by the time its value
// has been returned.
func (w *World) At(x, y int) Locator {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.get(x, y)
}

func (w *World) removeLocked(x, y int, update *[]Update) (orig interface{}) {
	defer func() { w.T(w, "removeLocked(%d,%d) = %v", x, y, orig) }()
	orig = w.Grid.Put(x, y, nil).(*Entity).Value()
	*update = append(*update, Update{x, y, nil})
	return orig
}

func (w *World) putEntityIfEmpty(x, y int, e *Entity, update *[]Update) (ok bool) {
	defer func() { w.T(w, "putEntityIfEmpty(%d,%d, %v) = %v", x, y, e, ok) }()

	dest := w.get(x, y)
	if !w.isEmpty(dest) {
		w.T(w, "isEmpty(%v) at %d,%d is false", dest, x, y)
		return false
	}
	dest.invalidate()
	w.Grid.Put(x, y, e)
	*update = append(*update, Update{x, y, &e.v})
	e.X = x
	e.Y = y
	return true
}

func (w *World) PutIfEmpty(x, y int, n interface{}) (loc Locator) {
	defer func() { w.T(w, "PutIfEmpty(%d,%d, %v) = %v", x, y, n, loc) }()
	w.mu.Lock()
	defer w.mu.Unlock()

	var update []Update
	loc = w.putIfEmptyLocked(x, y, n, &update)
	w.notify(update)
	return loc
}

func (w *World) putIfEmptyLocked(x, y int, n interface{}, update *[]Update) (loc Locator) {
	defer func() { w.T(w, "PutIfEmpty(%d,%d, %v) = %v", x, y, n, loc) }()
	e := w.createEntity(x, y, n)
	if w.putEntityIfEmpty(x, y, e, update) {
		return e
	}
	return nil
}

func (w *World) moveIfEmptyLocked(e *Entity, x, y int, update *[]Update) (ok bool) {
	defer func() { w.T(w, "moveIfEmptyLocked(%v, %d,%d) = %v", e, x, y, ok) }()
	ox, oy := e.X, e.Y
	if w.putEntityIfEmpty(x, y, e, update) {
		w.Grid.Put(ox, oy, nil)
		return true
	}
	return false
}

// PlaceRandomly places an occupant in a random location, and returns
// the (x, y) coordinates where it was placed.  The occupant will not
// be placed in a cell that's already occupied, unless the existing
// occupant is considered "empty" by the ConsiderEmpty callback.
func (w *World) PlaceRandomly(o interface{}) (loc Locator) {
	defer func() { w.T(w, "PlaceRandomly(%v) = %v", o, loc) }()
	for {
		x, y := rand.Intn(w.Width()), rand.Intn(w.Height())
		if loc := w.PutIfEmpty(x, y, o); loc != nil {
			w.T(o, "w.PlaceRandomly = %v", loc)
			return loc
		}
	}
}

// Copy returns a shallow copy of the world.
func (w *World) Copy() *World {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return &World{
		Grid:    *w.Grid.Copy(),
		EmptyFn: w.EmptyFn,
	}
}

// Each runs the given fn on each occupant in the world.
func (w *World) Each(fn func(loc Locator)) {
	w.mu.RLock()
	c := w.Grid.Copy()
	w.mu.RUnlock()

	c.Each(func(x, y int, o interface{}) {
		fn(o.(Locator))
	})
}

func (w *World) EachLocation(fn func(x, y int, o interface{})) {
	w.mu.RLock()
	c := w.Grid.Copy()
	w.mu.RUnlock()

	c.Each(func(x, y int, o interface{}) {
		fn(x, y, o.(Locator).Value())
	})
}

func (w *World) String() string {
	return fmt.Sprintf("[world %v]", &w.Grid)
}

func (s *World) T(e interface{}, msg string, args ...interface{}) {
	if s.Tracer != nil {
		a := []interface{}{e}
		a = append(a, args...)
		fmt.Fprintf(s.Tracer, fmt.Sprintf("%%v: %s\n", msg), a...)
	}
}

// New creates a World with the given dimensions.
func New(w, h int) *World {
	return &World{
		Grid: *NewGrid(w, h),
	}
}
