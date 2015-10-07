package grid2d

import "bytes"
import "encoding/gob"
import "fmt"
import "sync"

type PutWhenFunc func(existing, proposed interface{}) bool

var PutAlways PutWhenFunc = func(_, _ interface{}) bool {
	return true
}

var PutWhenNil PutWhenFunc = func(a, _ interface{}) bool {
	return a == nil
}

type Point struct {
	X, Y int
	V    interface{}
}

type Grid interface {
	Extents() (int, int)
	Get(x, y int) Locator
	Put(x, y int, n interface{}, fn PutWhenFunc) (interface{}, Locator)
	All() []Locator
	Locations() (int, int, []Point)
	Resize(width, height int, removedFn func(x, y int, o interface{}))

	Subscribe(ch chan<- []Update)
	Unsubscribe(ch chan<- []Update)
}

type grid struct {
	sync.RWMutex
	*notifier
	width, height int
	data          []*locator
}

func New(width, height int, done <-chan bool) Grid {
	return &grid{
		notifier: newNotifier(done),
		width:    width,
		height:   height,
		data:     make([]*locator, width*height),
	}
}

func (g *grid) Extents() (int, int) {
	g.RLock()
	defer g.RUnlock()
	return g.width, g.height
}

func (g *grid) offset(w, h int) int {
	if w < 0 || w > g.width || h < 0 || h > g.height {
		panic(fmt.Sprintf("grid index out of bounds: (%d,%d) is outside %dx%d", w, h, g.width, g.height))
	}
	return h*g.width + w
}

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

func (g *grid) Put(x, y int, n interface{}, fn PutWhenFunc) (interface{}, Locator) {
	g.Lock()
	defer g.Unlock()
	orig, loc := g.putLocked(x, y, n, fn)
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
	origLoc := g.getLocked(x, y)
	origValue := origLoc.Value()
	if !shouldPut(fn, origValue, n) {
		return origValue, nil
	}
	var loc *locator
	if n != nil {
		loc = &locator{g, x, y, n, false}
	}
	origLoc.invalidate()
	g.data[g.offset(x, y)] = loc
	if l, ok := n.(UsesLocator); ok {
		l.UseLocator(loc)
	}
	return origValue, loc
}

func (g *grid) moveLocked(x1, y1, x2, y2 int, fn PutWhenFunc) (interface{}, bool) {
	src := g.getLocked(x1, y1)
	dst := g.getLocked(x2, y2)
	if !shouldPut(fn, src.Value(), dst.Value()) {
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

func (g *grid) Locations() (int, int, []Point) {
	g.RLock()
	defer g.RUnlock()
	var points []Point
	for _, l := range g.data {
		if l != nil {
			points = append(points, Point{l.x, l.y, l.v})
		}
	}
	return g.width, g.height, points
}

func (g *grid) Resize(width, height int, removedFn func(x, y int, o interface{})) {
	g.Lock()
	defer g.Unlock()

	old := g.data
	g.data = make([]*locator, width*height)
	g.width = width
	g.height = height

	for _, l := range old {
		if l != nil {
			if l.x >= width || l.y >= height {
				removedFn(l.x, l.y, l.v)
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

func (g *grid) GobEncode() ([]byte, error) {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	width, height, locs := g.Locations()
	if err := enc.Encode(gobStruct{width, height, locs}); err != nil {
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
