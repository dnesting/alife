package grid2d

type PutWhenFunc func(existing, proposed interface{}) bool

var PutAlways PutWhenFunc = func(_, _ interface{}) bool {
	return true
}

var PutWhenNil PutWhenFunc = func(a, _ interface{}) bool {
	return a == nil
}

type Grid interface {
	Get(x, y int) Locator
	Put(x, y int, n interface{}, fn PutWhenFunc) (interface{}, Locator)
	All() []Locator
	Each(func(Locator))
}

type grid struct {
	sync.RWMutex
	w, h int
	data []Locator
}

func New(width, height int) Grid {
	return &grid{
		w:    width,
		h:    height,
		data: make([]Locator, w*h),
	}
}

func (g *grid) offset(w, h int) int {
	if w < 0 || w > g.w || h < 0 || h > g.h {
		panic(fmt.Sprintf("grid index out of bounds: (%d,%d) is outside %dx%d", w, h, g.w, g.h))
	}
	return h*g.w + w
}

func (g *grid) Get(x, y int) Locator {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.getLocked(x, y)
}

func (g *grid) getLocked(x, y int) Locator {
	return g.data[g.offset(x, y)]
}

func (g *grid) Put(x, y int, n interface{}, fn PutWhenFunc) (interface{}, Locator) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.putLocked(x, y, n, fn)
}

func shouldPut(fn PutWhenFunc, a, b interface{}) bool {
	if fn == nil {
		fn = PutAlways
	}
	return fn(a, b)
}

func (g *grid) putLocked(x, y int, n interface{}, fn PutWhenFunc) (interface{}, Locator) {
	origLoc := g.getLocked(x, y)
	origValue := origLoc.Value()
	if !shouldPut(fn, orig, n) {
		return origValue, nil
	}
	var loc Locator
	if n != nil {
		loc = &locator{x, y, n}
	}
	g.data[g.offset(x, y)] = loc
	origLoc.invalidate()
	return origValue, loc
}

func (g *grid) All() []Locator {
	g.mu.RLock()
	defer g.mu.RUnlock()
	var locs []Locator
	for _, l := range g.data {
		if l != nil {
			locs = append(locs, l)
		}
	}
	return locs
}
