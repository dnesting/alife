// Package world defines the world within which organisms and other occupants will exist.
package world

// See comments in entity.go for rules around holding mutexes to avoid deadlock.

import "fmt"
import "io"
import "math/rand"
import "sync"

// World is a place within which occupants can exist.  It contains various functions for
// retrieving and manipulating items by their (x, y) coordinates.  It is implemented as
// a toroidal 2D grid.
type World struct {
	multi    sync.RWMutex
	mu       sync.RWMutex
	Grid     Grid
	EmptyFn  func(o interface{}) bool
	UpdateFn func(w *World)
	Tracer   io.Writer
}

func (w *World) GobEncode() ([]byte, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.Grid.GobEncode()
}

func (w *World) GobDecode(stream []byte) error {
	return w.Grid.GobDecode(stream)
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

func (w *World) notifyUpdate() {
	if w.UpdateFn != nil {
		w.UpdateFn(w)
	}
}

func (w *World) validateCoords(x, y int) {
	if x >= w.Width() || y >= w.Height() || x < 0 || y < 0 {
		panic(fmt.Sprintf("(%d, %d) outside of world bounds (%d, %d)", x, y, w.Width(), w.Height()))
	}
}

func (w *World) createEntity(x, y int, mu *sync.Mutex, value interface{}) *Entity {
	w.validateCoords(x, y)
	if mu == nil {
		mu = &sync.Mutex{}
	}
	return &Entity{
		W:  w,
		X:  x,
		Y:  y,
		V:  value,
		mu: mu,
	}
}

func (w *World) putEntityLocked(x, y int, mu *sync.Mutex, value interface{}) (e *Entity) {
	defer func() { w.T(w, "putEntityLocked(%d,%d, %v, %v) = %v", x, y, mu, value, e) }()
	w.mu.Lock()
	defer w.mu.Unlock()
	e = w.createEntity(x, y, mu, value)
	w.Grid.Put(x, y, e)
	return e
}

// at returns the occupant at the given (x, y) coordinate.  Concurrent
// execution may mean that the occupant has moved by the time its value
// has been returned.
func (w *World) At(x, y int) Locator {
	w.mu.RLock()
	defer w.mu.RUnlock()
	l, _ := w.Grid.Get(x, y).(*Entity)
	return l
}

func (w *World) removeEntityLocked(x, y int) (orig interface{}) {
	defer func() { w.T(w, "removeEntityLocked(%d,%d) = %v", x, y, orig) }()
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.Grid.Put(x, y, nil).(*Entity).Value()
}

func (w *World) putEntityIfEmpty(x, y int, e *Entity) (ok bool) {
	defer func() { w.T(w, "putEntityIfEmpty(%d,%d, %v) = %v", x, y, e, ok) }()
	w.mu.Lock()
	defer w.mu.Unlock()

	dest := w.getWithEntityLockLocked(x, y)
	if dest != nil {
		defer dest.mu.Unlock()
	}
	if !w.isEmpty(dest) {
		w.T(w, "isEmpty(%v) at %d,%d is false", dest, x, y)
		return false
	}
	dest.invalidate()
	w.Grid.Put(x, y, e)
	e.X = x
	e.Y = y
	return true
}

func (w *World) PutIfEmpty(x, y int, n interface{}) (loc Locator) {
	defer func() { w.T(w, "PutIfEmpty(%d,%d, %v) = %v", x, y, n, loc) }()
	e := w.createEntity(x, y, nil, n)
	if w.putEntityIfEmpty(x, y, e) {
		return e
	}
	return nil
}

func (w *World) moveIfEmptyEntityLocked(e *Entity, x, y int) (ok bool) {
	defer func() { w.T(w, "moveIfEmptyEntityLocked(%v, %d,%d) = %v", e, x, y, ok) }()
	ox, oy := e.X, e.Y
	if w.putEntityIfEmpty(x, y, e) {
		w.Grid.Put(ox, oy, nil)
		return true
	}
	return false
}

func (w *World) getWithEntityLockLocked(x, y int) *Entity {
	o := w.Grid.Get(x, y)
	for {
		if o == nil {
			return nil
		}
		e := o.(*Entity)
		w.mu.Unlock()
		e.mu.Lock()
		w.mu.Lock()
		o := w.Grid.Get(x, y)
		if e.X != x || e.Y != y || e != o {
			e.mu.Unlock()
			continue
		}
		return e
	}
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

/*
func (w *World) withEntityLockLocked(x, y int, fn func(e *Entity)) {
	e := w.getWithEntityLockLocked(x, y)
	if e != nil {
		defer e.Unlock()
	}
	fn(e)
}

func (w *World) putLockedIgnoreExisting(x, y int, mu *sync.Mutex, value interface{}) *Entity {
	e := w.createEntity(x, y, mu, value)
	w.Grid.Put(x, y, e)
	return e
}

func (w *World) putEntityLocked(x, y int, mu *sync.Mutex, entity interface{}) *Entity {
	e := w.createEntity(x, y, mu, value)
	w.mu.Lock()
	defer w.mu.Unlock()
	w.putLockedIgnoreExisting(x, y, e)
	return e
}

func (w *World) Remove(x, y int) (removed interface{}) {
	defer func() { w.T(w, "Remove(%d,%d) = %v", x, y, removed) }()
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.removeLocked(x, y, false)
}

// Put places an occupant (or nil) in a cell, unconditionally.  The
// existing occupant, if any, is returned.
func (w *World) Put(x, y int, o interface{}) Locator {
	w.T(o, "w.Put(%d,%d)", x, y)
	defer w.notifyUpdate()
	w.mu.Lock()

	if old := w.get(x, y); old != nil {
		w.mu.Unlock()
		w.T(o, "- replacing %v", old)
		return old.Replace(o)
	}
	defer w.mu.Unlock()
	return w.putLocked(x, y, nil, o)
}

// clearIfEmpty clears (x,y) if possible and returns true, else returns false.
// Requires holding w.mu.
func (w *World) clearIfEmpty(x, y int) bool {
	// Concurrency rule (3) means we must give up w.mu before having the entity
	// being cleared remove itself.  Because this happens, there's a race where
	// another goroutine could do something with the entity in that cell, so we
	// enter a loop and have the entity double-check that it's in the same cell
	// we expect it to be in before removing it.
	count := 0
	w.validateCoords(x, y)
	for {
		old := w.get(x, y)
		if old == nil {
			// cell is empty, maintain w.mu and we'll just move pointers around
			break
		}

		if w.isEmpty(old.Value()) {
			// cell is occupied but needs to be replaced, so have the locator
			// remove itself without holding w.mu and we'll grab w.mu again
			// when it's done.
			w.mu.Unlock()
			removed := old.removeIfAt(x, y)
			w.mu.Lock()
			if removed {
				return true
			}
		} else {
			// cell is not empty, so fail the move operation
			return false
		}
		count++
		if count > 100 {
			panic(fmt.Sprintf("stuck trying to clear (%d,%d), most recently got %v: %v", x, y, old, old.Value()))
		}
	}
	return true
}

func (w *World) validateCoords(x, y int) {
	if x >= w.Width || y >= w.Height || x < 0 || y < 0 {
		panic(fmt.Sprintf("(%d, %d) outside of world bounds (%d, %d)", x, y, w.Width, w.Height))
	}
}

// PutIfEmpty places an occupant in a given location, but only if the
// location is empty, or considered "empty" by the ConsiderEmpty callback.
func (w *World) PutIfEmpty(x, y int, o interface{}) Locator {
	w.T(o, "w.PutIfEmpty(%d,%d)", x, y)
	defer w.notifyUpdate()
	w.mu.Lock()
	defer w.mu.Unlock()

	x, y = w.wrapCoords(x, y)

	if !w.clearIfEmpty(x, y) {
		return nil
	}

	l := w.putLockedIgnoreExisting(x, y, &sync.Mutex{}, o)
	w.T(o, "- %v", l)
	return l
}

// moveIfEmpty moves the given entity to its new coordinates.
// Requires that e.mu already be held.  If (x, y) is occupied but "empty",
// the entity there will be removed.  Returns true if the move occurred.
func (w *World) moveIfEmpty(e *Entity, x, y int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	x, y = w.wrapCoords(x, y)

	w.T(e, "clearIfEmpty(%d,%d)", x, y)
	if !w.clearIfEmpty(x, y) {
		w.T(e, "- not empty, returning false")
		return false
	}
	w.T(e, "- success, (%d,%d)=nil, (%d,%d)=self", e.X, e.Y, x, y)

	w.data[w.offset(e.X, e.Y)] = nil
	w.data[w.offset(x, y)] = e
	e.X, e.Y = x, y
	return true
}

*/

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
