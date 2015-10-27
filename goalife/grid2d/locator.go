// Mutations can be made to the Grid itself, but often it is more meaningful
// to make changes relative to an existing cell occupant.  This is accomplished
// through the use of Locators, which are returned for each occupant when it is
// placed in the Grid.
package grid2d

import "fmt"
import "os"
import "runtime"

// Locator is a handle that allows operations on the Grid relative to an
// occupant's position without requiring specific knowledge of the occupant's
// location, chiefly to avoid synchronization complexity on the part of the caller.
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

// UsesLocator can be implemented by occupant values if they want to be given a
// copy of their own Locator when they are placed in the Grid.  The Grid implementation
// will handle this automatically.
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

// delta returns the absolute coordinates given coordinates relative to the
// locator.
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

// Get retrieves the Locator of an occupant in a cell relative to the one currently
// referenced by this Locator.  loc.Get(0, 0) will thus return loc.  It is illegal to
// call this method on an invalidated Locator.
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

// Put places n in the Grid at the relative location dx,dy when fn returns true.
// Returns the occupant replaced, if any, and the Locator of the newly placed
// occupant, if it was placed.  It is illegal to call this method on an invalidated Locator.
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

// Move atomically changes the location of the Locator by dx,dy, provided fn
// returns true.  Returns the occupant replaced, if any, and a bool indicating
// whether a move occurred.  It is illegal to call this method on an invalidated
// Locator.
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

// Replace unconditionally replaces the occupant with n, and returns the
// new Locator for n.  The existing Locator is invalidated.  It is illegal to
// call this method on an invalidated Locator.
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

// Remove removes the occupant from the Grid, leaving its corresponding
// cell empty (nil).  The existing Locator is invalidated.  It is illegal to
// call this method on an invalidated Locator.
func (l *locator) Remove() {
	l.RemoveWithPlaceholder(l.v)
}

// RemoveWithPlaceholder removes the occupant from the Grid, and replaces
// the Locator's value with v.  While the removal is atomic, other
// goroutines may still have a reference to the Locator and may attempt to
// perform operations on its value, so this method permits specifying a replacement
// value so as to allow for reasonable future uses of the otherwise
// invalidated Locator.  It is illegal to call this method on an invalidated
// Locator.
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

// IsValid returns true if the Locator still references an occupant at a
// known location in the Grid.  A Locator is invalidated if it is replaced
// (such as by a call to Put, Replace or Move) or removed.  If the locator
// is nil, returns false.
func (l *locator) IsValid() bool {
	if l == nil {
		return false
	}
	l.w.RLock()
	defer l.w.RUnlock()
	return !l.invalid
}

// Value returns the occupant referenced by this Locator.  If the locator is
// nil, returns nil.
func (l *locator) Value() interface{} {
	if l != nil {
		return l.v
	}
	return nil
}
