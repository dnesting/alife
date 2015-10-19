package grid2d

import "bytes"
import "encoding/gob"
import "fmt"
import "math/rand"
import "sync"

import "github.com/dnesting/alife/goalife/log"

var Logger = log.Null()

type PutWhenFunc func(existing, proposed interface{}) bool

var PutAlways PutWhenFunc = func(_, _ interface{}) bool {
	return true
}

var PutWhenNil PutWhenFunc = func(a, _ interface{}) bool {
	Logger.Printf("PutWhenNil(%v) == nil? %v\n", a, a == nil)
	return a == nil
}

type Point struct {
	X, Y int
	V    interface{}
}

type NotifyStyle int

const (
	BufferFirst NotifyStyle = 1 << iota
	BufferLast
	BufferAll
	Unbuffered
)

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

	Subscribe(ch chan<- []Update, style NotifyStyle)
	Unsubscribe(ch chan<- []Update)
}

type grid struct {
	sync.RWMutex
	cond *sync.Cond
	*notifier
	width, height int
	data          []*locator
}

func New(width, height int, done <-chan bool, cond *sync.Cond) Grid {
	return &grid{
		cond:     cond,
		notifier: newNotifier(done),
		width:    width,
		height:   height,
		data:     make([]*locator, width*height),
	}
}

func (g *grid) String() string {
	return fmt.Sprintf("[grid %d,%d]", g.width, g.height)
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

func (g *grid) Remove(x, y int) interface{} {
	o, _ := g.Put(x, y, nil, PutAlways)
	return o
}

func (g *grid) Wait() {
	if g.cond != nil {
		g.cond.L.Lock()
		g.cond.Wait()
		g.cond.L.Unlock()
	}
}

func (g *grid) Put(x, y int, n interface{}, fn PutWhenFunc) (interface{}, Locator) {
	g.Lock()
	defer g.Unlock()
	return g.putLockedWithNotify(x, y, n, fn)
}

func (g *grid) PutRandomly(n interface{}, fn PutWhenFunc) (interface{}, Locator) {
	g.Lock()
	defer g.Unlock()
	var retry int
	for {
		orig, loc := g.putLockedWithNotify(rand.Intn(g.width), rand.Intn(g.height), n, fn)
		if loc != nil {
			return orig, loc
		}
		retry += 1
		if retry%10 == 0 {
			_, _, count := g.locationsLocked(nil)
			if count >= g.width*g.height {
				return nil, nil
			}
		}
	}
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

func (g *grid) Locations(points *[]Point) (int, int, int) {
	g.RLock()
	defer g.RUnlock()
	return g.locationsLocked(points)
}

func (g *grid) locationsLocked(points *[]Point) (int, int, int) {
	var count int
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
