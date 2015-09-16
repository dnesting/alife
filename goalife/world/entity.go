package world

// We aim for the world to be safe for concurrent access.  This requires setting some ground rules:
//
// 1. To access world.data[i] to obtain nil or an Entity, you must hold world.mu.
// 2. To modify or rely upon the location or content of an Entity, you must hold entity.mu.
// 3. It is permissible to perform (1) only after (2).  It is illegal to lock entity.mu while holding world.mu.
// 4. To do (2) with multiple entities at once, you must first hold world.multi.

import "fmt"
import "os"
import "runtime"
import "sync"

type Locator interface {
	Replace(n interface{}) Locator
	Relative(dx, dy int) Locator
	PutIfEmpty(dx, dy int, n interface{}) Locator
	MoveIfEmpty(dx, dy int) bool
	Remove()
	Value() interface{}
	WithLocation(fn func(x, y int, valid bool))
	Valid() bool
}

type Entity struct {
	mu      *sync.Mutex
	W       *World
	X, Y    int
	V       interface{}
	Invalid bool
	stack   []byte
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
	e.stack = make([]byte, 4096)
	runtime.Stack(e.stack, false)
}

func (e *Entity) Valid() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return !e.Invalid
}

func (e *Entity) checkValid() {
	if e.Invalid {
		fmt.Fprintf(os.Stderr, "%+v invalidated at:\n", e)
		os.Stderr.Write(e.stack)
		panic(fmt.Sprintf("access attempted to invalidated entity %+v", e))
	} else if e.X < 0 || e.Y < 0 {
		panic(fmt.Sprintf("invalid coordinates for valid entity (%d,%d)", e.X, e.Y))
	}
}

func (e *Entity) removeIfAt(x, y int) bool {
	e.W.T(e, "removeIfAt(%d,%d)", x, y)
	e.mu.Lock()
	defer e.mu.Unlock()
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

	e.W.mu.Lock()
	loc := e.W.put(e.X, e.Y, e.mu, n)
	e.W.mu.Unlock()

	e.W.T(e, "- with %v at (%d,%d)", loc, e.X, e.Y)
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

func (e *Entity) WithLocation(fn func(x, y int, valid bool)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	fn(e.X, e.Y, !e.Invalid)
}
