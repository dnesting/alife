// Package world defines the world within which organisms and other occupants will exist.
package world

// We aim for the world to be safe for concurrent access.  This requires setting some ground rules:
//
// 1. To access world.data[i] to obtain nil or an Entity, you must hold world.mu.
// 2. To modify or rely upon the location or content of an Entity, you must hold entity.mu.
// 3. It is permissible to perform (1) only after (2).  It is illegal to lock entity.mu while holding world.mu.
// 4. To do (2) with multiple entities at once, you must first hold world.multi.

import "bytes"
import "encoding/gob"
import "fmt"
import "io"
import "math/rand"
import "sync"

type Locator interface {
	Replace(n interface{}) Locator
	Relative(dx, dy int) Locator
	PutIfEmpty(dx, dy int, n interface{}) Locator
	MoveIfEmpty(dx, dy int) bool
	Remove()
	Value() interface{}
}

type Entity struct {
	mu      sync.Mutex
	W       *World
	X, Y    int
	V       interface{}
	Invalid bool
}

func (e *Entity) String() string {
	return fmt.Sprintf("(%d,%d)", e.X, e.Y)
}

func (e *Entity) invalidate() {
	if e == nil {
		return
	}
	e.X, e.Y = -1, -1
	e.Invalid = true
}

func (e *Entity) checkValid() {
	if e.Invalid {
		panic(fmt.Sprintf("access attempted to invalidated entity %+v", e))
	}
}

func (e *Entity) removeIfAt(x, y int) bool {
	e.W.T(e, "removeIfAt(%d,%d)", x, y)
	e.mu.Lock()
	defer e.mu.Unlock()
	e.checkValid()
	if e.X == x && e.Y == y {
		e.W.remove(x, y)
		return true
	}
	return false
}

func (e *Entity) checkLocationInvariant() {
	x := e.W.At(e.X, e.Y)
	if x != e {
		panic(fmt.Sprintf("inconsistent location: %v vs %v@(%d,%d)", e, x, e.X, e.Y))
	}
}

func (e *Entity) Remove() {
	e.W.T(e, "Remove")
	defer e.W.notifyUpdate()
	e.mu.Lock()
	defer e.mu.Unlock()
	e.checkValid()
	e.checkLocationInvariant()
	e.W.remove(e.X, e.Y)
}

func (e *Entity) Replace(n interface{}) Locator {
	e.W.T(e, "Replace(%v)", n)
	defer e.W.notifyUpdate()
	e.mu.Lock()
	defer e.mu.Unlock()
	e.checkValid()
	e.checkLocationInvariant()

	loc := e.W.put(e.X, e.Y, n)
	loc.checkLocationInvariant()
	return loc
}

func (e *Entity) Relative(dx, dy int) Locator {
	// Rule (3): e.w.At only needs the world lock.
	e.mu.Lock()
	defer e.mu.Unlock()
	e.checkValid()
	e.checkLocationInvariant()

	l := e.W.At(e.X+dx, e.Y+dy)
	e.W.T(e, "Relative(%d,%d) = %v", dx, dy, l)
	return l
}

func (e *Entity) PutIfEmpty(dx, dy int, n interface{}) Locator {
	// Rule (4): e.w.PutIfEmpty may end up replacing an existing
	// entity, so we need to grab the multi lock.
	e.W.multi.Lock()
	defer e.W.multi.Unlock()

	e.mu.Lock()
	defer e.mu.Unlock()
	e.checkValid()
	e.checkLocationInvariant()

	l := e.W.PutIfEmpty(e.X+dx, e.Y+dy, n)
	if l, ok := l.(*Entity); ok {
		l.checkLocationInvariant()
	}
	e.W.T(e, "PutIfEmpty(%d,%d, %v)", dx, dy, n)
	return l
}

func (e *Entity) MoveIfEmpty(dx, dy int) bool {
	defer e.W.notifyUpdate()
	// Rule (4): e.w.moveIfEmpty may end up replacing an existing
	// entity, so we need to grab the multi lock.
	e.W.multi.Lock()
	defer e.W.multi.Unlock()

	e.mu.Lock()
	defer e.mu.Unlock()
	e.checkValid()
	e.checkLocationInvariant()

	l := e.W.moveIfEmpty(e, e.X+dx, e.Y+dy)
	e.checkLocationInvariant()
	e.W.T(e, "MoveIfEmpty(%d,%d) = %v", dx, dy, l)
	return l
}

func (e *Entity) Value() interface{} {
	if e != nil {
		return e.V
	}
	return nil
}

// World is a place within which occupants can exist.  It contains various functions for
// retrieving and manipulating items by their (x, y) coordinates.  It is implemented as
// a toroidal 2D grid.
type World struct {
	Height, Width int

	multi    sync.RWMutex
	mu       sync.RWMutex
	data     []*Entity
	emptyFn  func(o interface{}) bool
	updateFn func(w *World)
	Tracer   io.Writer
}

func (w *World) GobEncode() ([]byte, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	if err := enc.Encode(w.Height); err != nil {
		return nil, err
	}
	if err := enc.Encode(w.Width); err != nil {
		return nil, err
	}
	if err := enc.Encode(w.data); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (w *World) GobDecode(stream []byte) error {
	dec := gob.NewDecoder(bytes.NewReader(stream))
	if err := dec.Decode(&w.Height); err != nil {
		return err
	}
	if err := dec.Decode(&w.Width); err != nil {
		return err
	}
	if err := dec.Decode(&w.data); err != nil {
		return err
	}
	return nil
}

// ConsiderEmpty allows the caller to specify a function used to determine if
// a cell is "sufficiently empty" for the purposes of the IfEmpty functions.
// This permits, for instance, placement of new organisms on top of food
// pellets, while not permitting movement of organisms into the same cells.
// The function should return true if the occupant of the cell can be
// considered empty enough.
func (w *World) ConsiderEmpty(fn func(o interface{}) bool) {
	w.emptyFn = fn
}

// OnUpdate specifies a func to be called every time a change to the world
// occurs. Changes include placement, removal, or movement of an occupant.
func (w *World) OnUpdate(fn func(w *World)) {
	w.updateFn = fn
}

func (w *World) isEmpty(o interface{}) bool {
	if o == nil {
		return true
	}
	if w.emptyFn == nil {
		return false
	}
	return w.emptyFn(o)
}

// Dimensions gives the width and height dimensions of the world.
func (w *World) Dimensions() (int, int) {
	return w.Width, w.Height
}

func modPos(v, max int) int {
	v %= max
	if v < 0 {
		v += max
	}
	return v
}

// wrapCoords ensures the x and y coordinates are in-bounds
func (w *World) wrapCoords(x, y int) (int, int) {
	x = modPos(x, w.Width)
	y = modPos(y, w.Height)

	return x, y
}

// offset converts the (x, y) coordinates to a slice offset.  The given
// coordinates can be outside of the (width, height) ranges for the world,
// which will just result in the location wrapping around.
func (w *World) offset(x, y int) int {
	x, y = w.wrapCoords(x, y)
	return modPos(y*w.Width+x, w.Height*w.Width)
}

func (w *World) notifyUpdate() {
	if w.updateFn != nil {
		w.updateFn(w)
	}
}

func (w *World) get(x, y int) *Entity {
	return w.data[w.offset(x, y)]
}

func (w *World) put(x, y int, entity interface{}) *Entity {
	w.validateCoords(x, y)
	e := &Entity{
		W: w,
		X: x,
		Y: y,
		V: entity,
	}
	o := w.offset(x, y)
	w.data[o].invalidate()
	w.data[o] = e
	return e
}

func (w *World) remove(x, y int) interface{} {
	w.mu.Lock()
	defer w.mu.Unlock()
	o := w.offset(x, y)
	old := w.data[o]
	w.data[o] = nil
	old.invalidate()
	return old.Value()
}

// at returns the occupant at the given (x, y) coordinate.  Concurrent
// execution may mean that the occupant has moved by the time its value
// has been returned.
func (w *World) At(x, y int) Locator {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.get(x, y)
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
	return w.put(x, y, o)
}

// PlaceRandomly places an occupant in a random location, and returns
// the (x, y) coordinates where it was placed.  The occupant will not
// be placed in a cell that's already occupied, unless the existing
// occupant is considered "empty" by the ConsiderEmpty callback.
func (w *World) PlaceRandomly(o interface{}) Locator {
	width, height := w.Dimensions()
	for {
		x, y := rand.Intn(width), rand.Intn(height)
		if loc := w.PutIfEmpty(x, y, o); loc != nil {
			w.T(o, "w.PlaceRandomly = %v", loc)
			return loc
		}
	}
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

	l := w.put(x, y, o)
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

// Copy returns a shallow copy of the world.
func (w *World) Copy() *World {
	w.mu.RLock()
	defer w.mu.RUnlock()

	data := make([]*Entity, w.Height*w.Width)
	copy(data, w.data)

	return &World{
		Height:  w.Height,
		Width:   w.Width,
		data:    data,
		emptyFn: w.emptyFn,
	}
}

// Each runs the given fn on each occupant in the world.
func (w *World) Each(fn func(loc Locator)) {
	c := w.Copy()
	for i := 0; i < len(c.data); i++ {
		if c.data[i] != nil {
			fn(c.data[i])
		}
	}
}

// Printable is implemented by occupants that want to control how they are
// presented on terminal-based renderings of the world.
type Printable interface {
	Rune() rune
}

func (w *World) String() string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	b := bytes.Buffer{}
	headFoot := func() {
		b.WriteString("+")
		for x := 0; x < w.Width; x++ {
			b.WriteString("-")
		}
		b.WriteString("+\n")
	}

	headFoot()
	for y := 0; y < w.Height; y++ {
		b.WriteString("|")
		for x := 0; x < w.Width; x++ {
			i := w.data[w.offset(x, y)]
			if i == nil {
				b.WriteString(" ")
			} else {
				switch i := i.Value().(type) {
				case Printable:
					b.WriteRune(i.Rune())
				default:
					b.WriteString("?")
				}
			}
		}
		b.WriteString("|\n")
	}
	headFoot()
	return b.String()
}

// New creates a World with the given dimensions.
func New(h, w int) *World {
	return &World{
		Height: h,
		Width:  w,
		data:   make([]*Entity, h*w),
	}
}

func (s *World) T(e interface{}, msg string, args ...interface{}) {
	if s.Tracer != nil {
		a := []interface{}{e}
		a = append(a, args...)
		fmt.Fprintf(s.Tracer, fmt.Sprintf("%%v: %s\n", msg), a...)
	}
}
