package grid2d

import "fmt"
import "os"
import "runtime"

type Locator interface {
	Get(dx, dy int) Locator
	Put(dx, dy int, n interface{}, fn PutWhenFunc) (interface{}, Locator)
	Move(dx, dy int, fn PutWhenFunc) (interface{}, bool)
	Replace(n interface{}) Locator
	Remove()
	RemoveWithPlaceholder(v interface{})
	IsValid() bool
	Value() interface{}
}

type UsesLocator interface {
	UseLocator(loc Locator)
}

type locator struct {
	w        *grid
	x, y     int
	v        interface{}
	invalid  bool
	invStack []byte
}

func newLocator(w *grid, x, y int, v interface{}) *locator {
	return &locator{
		w:        w,
		x:        x,
		y:        y,
		v:        v,
		invStack: make([]byte, 8192),
	}
}

func (l *locator) String() string {
	invalid := ""
	if l.invalid {
		invalid = " invalid"
	}
	return fmt.Sprintf("[%d,%d%s]", l.x, l.y, invalid)
}

func (l *locator) checkValid() {
	if l.invalid {
		fmt.Fprintf(os.Stderr, "invalidated at:\n")
		os.Stderr.Write(l.invStack)
		fmt.Fprintln(os.Stderr)
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
	if loc := l.w.getLocked(l.delta(dx, dy)); loc != nil {
		return loc
	}
	return nil
}

func (l *locator) Put(dx, dy int, n interface{}, fn PutWhenFunc) (interface{}, Locator) {
	l.w.Lock()
	l.checkValid()
	l.checkLocationInvariant()
	x, y := l.delta(dx, dy)
	orig, loc := l.w.putLocked(x, y, n, fn)
	if loc != nil {
		l.w.RecordAdd(x, y, n)
		l.w.Unlock()
		l.w.Wait()
		return orig, loc
	}
	l.w.Unlock()
	return orig, nil
}

func (l *locator) Move(dx, dy int, fn PutWhenFunc) (interface{}, bool) {
	l.w.Lock()
	l.checkValid()
	l.checkLocationInvariant()
	x2, y2 := l.delta(dx, dy)

	orig, ok := l.w.moveLocked(l.x, l.y, x2, y2, fn)
	l.checkValid()
	l.checkLocationInvariant()

	l.w.Unlock()

	if ok {
		l.w.Wait()
	}

	return orig, ok
}

func (l *locator) Replace(n interface{}) Locator {
	l.w.Lock()
	loc := l.replaceLocked(n)
	l.w.Unlock()
	if loc != nil {
		l.w.Wait()
	}
	return loc
}

func (l *locator) replaceLocked(n interface{}) Locator {
	l.checkValid()
	l.checkLocationInvariant()
	old := l.v
	if _, loc := l.w.putLocked(l.x, l.y, n, PutAlways); loc != nil {
		if n == nil {
			l.w.RecordRemove(l.x, l.y, old)
		} else {
			l.w.RecordReplace(l.x, l.y, old, n)
		}
		return loc
	}
	return nil
}

func (l *locator) Remove() {
	l.RemoveWithPlaceholder(l.v)
}

func (l *locator) RemoveWithPlaceholder(v interface{}) {
	l.w.Lock()
	if l.invalid {
		l.w.Unlock()
		return
	}
	l.checkLocationInvariant()
	l.replaceLocked(nil)
	l.v = v
	l.w.Unlock()
	l.w.Wait()
}

func (l *locator) invalidate() {
	if l == nil {
		return
	}
	l.invalid = true
	runtime.Stack(l.invStack, false)
}

func (l *locator) IsValid() bool {
	if l == nil {
		return false
	}
	l.w.RLock()
	defer l.w.RUnlock()
	return !l.invalid
}

func (l *locator) Value() interface{} {
	if l != nil {
		return l.v
	}
	return nil
}
