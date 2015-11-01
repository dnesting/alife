// Package grid2d provides a 2D discrete-cell "world" within which
// occupants may live their lives.
package grid2d

import "bytes"
import "encoding/gob"
import "fmt"
import "math/rand"
import "sync"

import "github.com/dnesting/alife/goalife/log"

var Logger = log.Null()

// PutWhenFunc is called any time an occupant will be placed in the Grid to
// establish whether or not the Put should proceed depending on the contents
// of the cell.
type PutWhenFunc func(existing, proposed interface{}) bool

// PutAlways signals that a Put should always proceed, regardless of whether
// the cell is already occupied.
var PutAlways PutWhenFunc = func(_, _ interface{}) bool {
	return true
}

// PutWhenNil signals that a Put should only succeed if the cell value is
// nil (the cell is empty).
var PutWhenNil PutWhenFunc = func(a, _ interface{}) bool {
	return a == nil
}

// Point is a value located at a specific coordinate in the Grid.
type Point struct {
	X, Y int
	V    interface{}
}

// Grid is a 2D world that holds occupants at specific discrete coordinates.
type Grid interface {
	Extents() (int, int)
	Get(x, y int) Locator
	Put(x, y int, n interface{}, fn PutWhenFunc) (interface{}, Locator)
	PutRandomly(n interface{}, fn PutWhenFunc) (interface{}, Locator)
	Remove(x, y int) interface{}
	All() []Locator
	Locations(points *[]Point) (int, int, int)
	Resize(width, height int, removedFn func(x, y int, o interface{}))
	Wait()

	Subscribe(ch chan<- []Update)
	Unsubscribe(ch chan<- []Update)
	CloseSubscribers()
}

type grid struct {
	sync.RWMutex
	cond *sync.Cond
	notifier

	width, height int
	data          []*locator
}

// New creates a Grid with the given extents.
//
// If cond is provided, every world-mutating operation will call
// cond.Wait to ensure events are synchronized.  This is useful to
// synchronize updates with rendering.
func New(width, height int, cond *sync.Cond) Grid {
	return &grid{
		cond:   cond,
		width:  width,
		height: height,
		data:   make([]*locator, width*height),
	}
}

func (g *grid) String() string {
	return fmt.Sprintf("[grid %d,%d]", g.width, g.height)
}

// Extents returns the size of the world.
func (g *grid) Extents() (width int, height int) {
	g.RLock()
	defer g.RUnlock()
	return g.width, g.height
}

// offset converts x,y coordinates to the g.data offset for that cell.
func (g *grid) offset(x, y int) int {
	if x < 0 || x > g.width || y < 0 || y > g.height {
		panic(fmt.Sprintf("grid index out of bounds: (%d,%d) is outside %dx%d", x, y, g.width, g.height))
	}
	return y*g.width + x
}

// Get retrieves the Locator for any occupant at x,y.  If the cell is
// empty, returns nil.
func (g *grid) Get(x, y int) Locator {
	g.RLock()
	defer g.RUnlock()
	if loc := g.getLocked(x, y); loc != nil {
		return loc
	}
	return nil
}

func (g *grid) getLocked(x, y int) *locator {
	return g.data[g.offset(x, y)]
}

// Remove removes any occupant at x,y, and returns it.
func (g *grid) Remove(x, y int) interface{} {
	o, _ := g.Put(x, y, nil, PutAlways)
	return o
}

// Wait invokes Wait on the sync.Cond object provided during initialization.
// This method is a no-op if no Cond was provided.
func (g *grid) Wait() {
	if g.cond != nil {
		g.cond.L.Lock()
		g.cond.Wait()
		g.cond.L.Unlock()
	}
}

// Put places n at x,y when fn returns true.  Returns the existing occupant,
// and a Locator instance that can be used to relate n to the grid in the future.
func (g *grid) Put(x, y int, n interface{}, fn PutWhenFunc) (interface{}, Locator) {
	g.Lock()
	defer g.Unlock()
	return g.putLockedWithNotify(x, y, n, fn)
}

// PutRandomly places n at a random location in the grid.  Returns any occupant
// that was replaced, and a Locator instance that can be used to relate n to the grid
// in the future.  If no open cells are available, returns nil, nil.
func (g *grid) PutRandomly(n interface{}, fn PutWhenFunc) (interface{}, Locator) {
	g.Lock()
	defer g.Unlock()

	offsets := rand.Perm(len(g.data))
	for _, offset := range offsets {
		x, y := offset%g.width, offset/g.width
		orig, loc := g.putLockedWithNotify(x, y, n, fn)
		if loc != nil {
			return orig, loc
		}
	}
	return nil, nil
}

func (g *grid) putLockedWithNotify(x, y int, n interface{}, fn PutWhenFunc) (interface{}, Locator) {
	orig, loc := g.putLocked(x, y, n, fn)
	if orig != nil && n == nil {
		g.RecordRemove(x, y, orig)
	}
	if loc != nil {
		g.RecordAdd(x, y, n)
		return orig, loc
	}
	return orig, nil
}

func shouldPut(fn PutWhenFunc, a, b interface{}) bool {
	if fn == nil {
		fn = PutAlways
	}
	return fn(a, b)
}

func (g *grid) putLocked(x, y int, n interface{}, fn PutWhenFunc) (interface{}, *locator) {
	Logger.Printf("%v.putLocked(%d,%d, %v)\n", g, x, y, n)
	origLoc := g.getLocked(x, y)
	origValue := origLoc.Value()
	if !shouldPut(fn, origValue, n) {
		return origValue, nil
	}
	var loc *locator
	if n != nil {
		loc = newLocator(g, x, y, n)
	}
	origLoc.invalidate()
	g.data[g.offset(x, y)] = loc
	if l, ok := n.(UsesLocator); ok {
		l.UseLocator(loc)
	}
	return origValue, loc
}

func (g *grid) moveLocked(x1, y1, x2, y2 int, fn PutWhenFunc) (interface{}, bool) {
	Logger.Printf("%v.moveLocked(%d,%d, %d,%d)\n", g, x1, y1, x2, y2)
	src := g.getLocked(x1, y1)
	dst := g.getLocked(x2, y2)
	if !shouldPut(fn, dst.Value(), src.Value()) {
		return dst.Value(), false
	}
	dst.invalidate()
	g.data[g.offset(x2, y2)] = src
	g.data[g.offset(x1, y1)] = nil
	src.x = x2
	src.y = y2
	g.RecordMove(x1, y1, x2, y2, src.v)
	return dst.Value(), true
}

// All returns the Locators for all occupants in the grid.
func (g *grid) All() []Locator {
	g.RLock()
	defer g.RUnlock()
	var locs []Locator
	for _, l := range g.data {
		if l != nil {
			locs = append(locs, l)
		}
	}
	return locs
}

// Locations populates points with all occupants in the grid.  Returns the width
// and height of the Grid at the time along with a count of the number of occupants
// found.  Points may be nil if you just want to get a quick count.
func (g *grid) Locations(points *[]Point) (width int, height int, count int) {
	g.RLock()
	defer g.RUnlock()
	return g.locationsLocked(points)
}

func (g *grid) locationsLocked(points *[]Point) (width int, height int, count int) {
	if points != nil {
		if cap(*points) < g.width*g.height {
			*points = make([]Point, 0, g.width*g.height)
		}
		*points = (*points)[:0]
	}
	for _, l := range g.data {
		if l != nil {
			if points != nil {
				*points = append(*points, Point{l.x, l.y, l.v})
			}
			count++
		}
	}
	return g.width, g.height, count
}

// Resize changes the dimensions of the Grid.  Any occupants that find themselves
// outside of the resized Grid are passed individually to removedFn before being
// discarded.
func (g *grid) Resize(width, height int, removedFn func(x, y int, o interface{})) {
	Logger.Printf("%g.Resize(%d,%d)\n", width, height)
	g.Lock()
	defer g.Unlock()

	old := g.data
	g.data = make([]*locator, width*height)
	g.width = width
	g.height = height

	for _, l := range old {
		if l != nil {
			if l.x >= width || l.y >= height {
				if removedFn != nil {
					removedFn(l.x, l.y, l.v)
				}
				g.RecordRemove(l.x, l.y, l.v)
			} else {
				g.data[g.offset(l.x, l.y)] = l
			}
		}
	}
}

type gobStruct struct {
	Width  int
	Height int
	Points []Point
}

var gobData gobStruct

func (g *grid) GobEncode() ([]byte, error) {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	width, height, _ := g.Locations(&gobData.Points)
	gobData.Width = width
	gobData.Height = height
	if err := enc.Encode(gobData); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (g *grid) GobDecode(data []byte) error {
	b := bytes.NewBuffer(data)
	dec := gob.NewDecoder(b)
	var gs gobStruct
	if err := dec.Decode(&gs); err != nil {
		return err
	}
	g.Resize(gs.Width, gs.Height, nil)
	for _, p := range gs.Points {
		g.Put(p.X, p.Y, p.V, PutAlways)
	}
	return nil
}
