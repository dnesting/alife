package world

// We aim for the world to be safe for concurrent access.  This requires setting some ground rules:
//
// 1. To access world.data[i] to obtain nil or an Entity, you must hold world.mu.
// 2. To modify or rely upon the location or content of an Entity, you must hold entity.mu.
// 3. It is permissible to perform (1) only after (2).  It is illegal to lock entity.mu while holding world.mu.
// 4. To do (2) with multiple entities at once, you must first hold world.multi.

import "bytes"
import "fmt"
import "encoding/gob"
import "runtime"
import "sync"

type Locator interface {
	Replace(n interface{}) Locator
	Relative(dx, dy int) Locator
	PutIfEmpty(dx, dy int, n interface{}) Locator
	MoveIfEmpty(dx, dy int) bool
	Remove()
	Value() interface{}
	Valid() bool
}

type Locatable interface {
	UseLocator(l Locator)
}

type Entity struct {
	// These are protected by W.mu
	X, Y int
	v    interface{}
	w    *World

	mu      sync.Mutex
	Invalid bool
	stack   []byte
}

func (e *Entity) GobEncode() ([]byte, error) {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	if err := enc.Encode(e.X); err != nil {
		return nil, err
	}
	if err := enc.Encode(e.Y); err != nil {
		return nil, err
	}
	if err := enc.Encode(&e.v); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (e *Entity) GobDecode(stream []byte) error {
	dec := gob.NewDecoder(bytes.NewReader(stream))
	if err := dec.Decode(&e.X); err != nil {
		return err
	}
	if err := dec.Decode(&e.Y); err != nil {
		return err
	}
	if err := dec.Decode(&e.v); err != nil {
		return err
	}
	if l, ok := e.v.(Locatable); ok {
		l.UseLocator(e)
	}
	return nil
}

func (e *Entity) UseWorld(w *World) {
	e.w = w
}

func (e *Entity) String() string {
	return fmt.Sprintf("(%d,%d)", e.X, e.Y)
}

func (e *Entity) invalidate() {
	if e == nil {
		return
	}
	e.Invalid = true
	e.stack = make([]byte, 4096)
	runtime.Stack(e.stack, false)
}

func (e *Entity) Valid() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return !e.Invalid
}

/*
func (e *Entity) checkValidLocked() {
	if e.Invalid {
		fmt.Fprintf(os.Stderr, "%+v invalidated at:\n", e)
		os.Stderr.Write(e.stack)
		panic(fmt.Sprintf("access attempted to invalidated entity %+v", e))
	} else if e.X < 0 || e.Y < 0 {
		panic(fmt.Sprintf("invalid coordinates for valid entity (%d,%d)", e.X, e.Y))
	}
}
*/

/*
func (e *Entity) removeIfAt(x, y int) bool {
	e.w.T(e, "removeIfAt(%d,%d)", x, y)
	if e.X == x && e.Y == y {
		old := e.w.removeLocked(x, y)
		if old != e.Value() {
			panic(fmt.Sprintf("removed (%d,%d) entity %v, expected %v", x, y, old, e))
		}
		e.invalidate()
		return true
	}
	return false
}
*/

func (e *Entity) checkLocationInvariant() {
	x := e.w.get(e.X, e.Y)
	if x != e {
		panic(fmt.Sprintf("inconsistent location: %v vs %v@(%d,%d)", e, x, e.X, e.Y))
	}
}

func (e *Entity) Remove() {
	e.w.T(e, "Remove")
	defer e.w.notifyUpdate()
	e.w.mu.Lock()
	defer e.w.mu.Unlock()
	if !e.Invalid {
		e.checkLocationInvariant()
		e.w.removeLocked(e.X, e.Y)
		e.invalidate()
	}
}

func (e *Entity) Replace(n interface{}) Locator {
	e.w.T(e, "Replace(%v)", n)
	defer e.w.notifyUpdate()
	e.w.mu.Lock()
	defer e.w.mu.Unlock()
	if !e.Invalid {
		e.checkLocationInvariant()

		ne := e.w.putLocked(e.X, e.Y, n)

		e.w.T(e, "- with %v at (%d,%d)", ne, e.X, e.Y)
		ne.checkLocationInvariant()
		return ne
	}
	return nil
}

func (e *Entity) Relative(dx, dy int) Locator {
	// Rule (3): e.w.Get only needs the world lock.
	e.w.mu.Lock()
	defer e.w.mu.Unlock()
	if !e.Invalid {
		e.checkLocationInvariant()
		x, y := e.w.Wrap(e.X+dx, e.Y+dy)
		l := e.w.get(x, y)
		e.w.T(e, "Relative(%d,%d) = %v", dx, dy, l)
		return l
	}
	return nil
}

func (e *Entity) PutIfEmpty(dx, dy int, n interface{}) Locator {
	defer e.w.notifyUpdate()
	// Rule (4): e.w.PutIfEmpty may end up replacing an existing
	// entity, so we need to grab the multi lock.
	e.w.mu.Lock()
	defer e.w.mu.Unlock()

	if !e.Invalid {
		e.checkLocationInvariant()

		x, y := e.w.Wrap(e.X+dx, e.Y+dy)
		l := e.w.putIfEmptyLocked(x, y, n)
		if l, ok := l.(*Entity); ok {
			l.checkLocationInvariant()
		}
		e.w.T(e, "PutIfEmpty(%d,%d, %v)", dx, dy, n)
		return l
	}
	return nil
}

func (e *Entity) MoveIfEmpty(dx, dy int) bool {
	defer e.w.notifyUpdate()
	// Rule (4): e.w.moveIfEmpty may end up replacing an existing
	// entity, so we need to grab the multi lock.
	e.w.mu.Lock()
	defer e.w.mu.Unlock()

	if !e.Invalid {
		e.checkLocationInvariant()

		x, y := e.w.Wrap(e.X+dx, e.Y+dy)

		ok := e.w.moveIfEmptyLocked(e, x, y)
		e.checkLocationInvariant()
		e.w.T(e, "MoveIfEmpty(%d,%d) = %v", dx, dy, ok)
		return ok
	}
	return false
}

func (e *Entity) Value() interface{} {
	if e != nil {
		return e.v
	}
	return nil
}
