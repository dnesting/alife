// Package world defines the world within which organisms and other occupants will exist.
package world

import "bytes"
import "encoding/gob"
import "fmt"
import "math/rand"
import "sync"

// Occupant is anything that occupies a cell in a world.
type Occupant interface{}

// World is a place within which occupants can exist.  It contains various functions for
// retrieving and manipulating items by their (x, y) coordinates.  It is implemented as
// a toroidal 2D grid.
type World struct {
	Height, Width int

	mu       sync.RWMutex
	data     []Occupant
	emptyFn  func(o Occupant) bool
	updateFn func(w *World)
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
func (w *World) ConsiderEmpty(fn func(o Occupant) bool) {
	w.emptyFn = fn
}

// OnUpdate specifies a func to be called every time a change to the world
// occurs. Changes include placement, removal, or movement of an occupant.
func (w *World) OnUpdate(fn func(w *World)) {
	w.updateFn = fn
}

func (w *World) isEmpty(o Occupant) bool {
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

// At returns the occupant at the given (x, y) coordinate.  Concurrent
// execution may mean that the occupant has moved by the time its value
// has been returned.
func (w *World) At(x, y int) Occupant {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.data[w.offset(x, y)]
}

// Put places an occupant (or nil) in a cell, unconditionally.  The
// existing occupant, if any, is returned.
func (w *World) Put(x, y int, o Occupant) Occupant {
	defer w.notifyUpdate()
	w.mu.Lock()
	defer w.mu.Unlock()

	offset := w.offset(x, y)
	old := w.data[offset]
	w.data[offset] = o
	return old
}

// PlaceRandomly places an occupant in a random location, and returns
// the (x, y) coordinates where it was placed.  The occupant will not
// be placed in a cell that's already occupied, unless the existing
// occupant is considered "empty" by the ConsiderEmpty callback.
func (w *World) PlaceRandomly(o Occupant) (int, int) {
	width, height := w.Dimensions()
	var x, y int
	for {
		x, y = rand.Intn(width), rand.Intn(height)
		if w.PutIfEmpty(x, y, o) == nil {
			break
		}
	}
	return x, y
}

// PutIfEmpty places an occupant in a given location, but only if the
// location is empty, or considered "empty" by the ConsiderEmpty callback.
func (w *World) PutIfEmpty(x, y int, o Occupant) Occupant {
	defer w.notifyUpdate()
	w.mu.Lock()
	defer w.mu.Unlock()

	offset := w.offset(x, y)
	if w.isEmpty(w.data[offset]) {
		w.data[offset] = o
		return nil
	}
	return w.data[offset]
}

// MoveIfEmpty moves the occupant at (x1, y1) into the location (x2, y2),
// but only if the location is empty, or considered "empty" by the
// ConsiderEmpty callback.
func (w *World) MoveIfEmpty(x1, y1, x2, y2 int) Occupant {
	defer w.notifyUpdate()
	w.mu.Lock()
	defer w.mu.Unlock()

	o1 := w.offset(x1, y1)
	o2 := w.offset(x2, y2)

	// If source cell is empty, don't move anything (return success)
	if w.data[o1] == nil {
		return nil
	}
	// If dest cell isn't empty, don't move anything (return its occupant)
	if w.data[o2] != nil {
		return w.data[o2]
	}
	w.data[o2] = w.data[o1]
	w.data[o1] = nil
	return nil
}

// RemoveIfEqual removes the given occupant from the given (x, y) coordinates.
// The occupant is only removed if the occupant at (x, y) is the same as the
// one passed.
func (w *World) RemoveIfEqual(x, y int, o Occupant) Occupant {
	return w.ReplaceIfEqual(x, y, o, nil)
}

// ReplaceIfEqual removes the given occupant o from the given (x, y) coordinates,
// and replaces it with the given occupant n (which may be nil). Replacement only
// occurs if the occupant at (x, y) is the same as o.
func (w *World) ReplaceIfEqual(x, y int, o Occupant, n Occupant) Occupant {
	defer w.notifyUpdate()
	w.mu.Lock()
	defer w.mu.Unlock()

	offset := w.offset(x, y)
	orig := w.data[offset]
	if orig == o {
		w.data[offset] = n
	}
	return orig
}

// Remove removes the occupant at (x, y), and returns it.
func (w *World) Remove(x, y int) Occupant {
	return w.Put(x, y, nil)
}

// Copy returns a shallow copy of the world.
func (w *World) Copy() *World {
	w.mu.RLock()
	defer w.mu.RUnlock()

	data := make([]Occupant, w.Height*w.Width)
	copy(data, w.data)

	return &World{
		Height:  w.Height,
		Width:   w.Width,
		data:    data,
		emptyFn: w.emptyFn,
	}
}

// Each runs the given fn on each occupant in the world.
func (w *World) Each(fn func(x, y int, o Occupant)) {
	for y := 0; y < w.Height; y++ {
		for x := 0; x < w.Width; x++ {
			if i := w.At(x, y); i != nil {
				fn(x, y, i)
			}
		}
	}
}

func (w *World) String() string {
	return fmt.Sprintf("[world %dx%d]", w.Width, w.Height)
}

// New creates a World with the given dimensions.
func New(h, w int) *World {
	return &World{
		Height: h,
		Width:  w,
		data:   make([]Occupant, h*w),
	}
}
