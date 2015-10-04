package grid2d

type Locator interface {
	Get(dx, dy int) Locator
	Put(dx, dy int, n interface{}, fn PutWhenFunc) (interface{}, Locator)
	Move(dx, dy int, fn PutWhenFunc) (interface{}, bool)
	Replace(n interface{}) Locator
	Remove()
	IsValid() bool
}

type locator struct {
	w       *grid
	x, y    int
	v       interface{}
	invalid bool
}

func (l *locator) checkValid() {
	if l.invalid {
		panic("attempt to use an invalidated locator")
	}
}

func (l *locator) checkLocationInvariant() {
	found := l.w.getLocked(l.x, l.y)
	if l != found {
		panic(fmt.Sprintf("inconsistent location: %v versus %v found at (%d,%d)", l, found, l.x, l.y))
	}
}

func (l *locator) delta(dx, dy int) (int, int) {
	x := (l.x + dx) % l.w.width
	y := (l.y + dy) % l.w.height
	if x < 0 {
		x += l.w.width
	}
	if y < 0 {
		y += l.w.height
	}
	return x, y
}

func (l *locator) Get(dx, dy int) Locator {
	l.w.RLock()
	defer l.w.RUnlock()
	l.checkValid()
	l.checkLocationInvariant()
	return l.w.getLocked(l.delta(dx, dy))
}

func (l *locator) Put(dx, dy int, n interface{}, fn PutWhenFunc) (interface{}, Locator) {
	l.w.Lock()
	defer l.w.Unlock()
	l.checkValid()
	l.checkLocationInvariant()
	x, y := l.delta(dx, dy)
	return l.w.putLocked(x, y, n, fn)
}

func (l *locator) Move(dx, dy int, fn PutWhenFunc) (interface{}, bool) {
	l.w.Lock()
	defer l.w.Unlock()
	l.checkValid()
	l.checkLocationInvariant()
	x2, y2 := l.delta(dx, dy)

	orig, ok := l.w.moveLocked(l.x, l.y, x2, y2, fn)
	l.checkValid()
	l.checkLocationInvariant()

	return orig, ok
}

func (l *locator) Replace(n interface{}) Locator {
	l.w.Lock()
	defer l.w.Unlock()
	l.checkValid()
	l.checkLocationInvariant()
	return l.w.putLocked(l.x, l.y, n, PutAlways)
}

func (l *locator) Remove() Locator {
	l.w.Lock()
	defer l.w.Unlock()
	if l.invalid {
		return
	}
	l.checkLocationInvariant()
	return l.Replace(nil)
}

func (l *locator) invalidate() {
	l.w.Lock()
	defer l.w.Unlock()
	l.invalid = true
}

func (l *locator) IsValid() {
	l.w.RLock()
	defer l.w.RUnlock()
	return l.invalid
}
