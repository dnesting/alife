package grid2d

import "testing"

// type Grid interface {
// 	Extents() (int, int)
// 	Get(x, y int) Locator
// 	Put(x, y int, n interface{}, fn PutWhenFunc) (interface{}, Locator)
// 	All() []Locator
// }

// type Locator interface {
// 	Get(dx, dy int) Locator
// 	Put(dx, dy int, n interface{}, fn PutWhenFunc) (interface{}, Locator)
// 	Move(dx, dy int, fn PutWhenFunc) (interface{}, bool)
// 	Replace(n interface{}) Locator
// 	Remove()
// 	IsValid() bool
// 	Value() interface{}
// }

func TestGetPut(t *testing.T) {
	g := New(3, 3, nil)
	_, l := g.Put(1, 1, 11, PutAlways)

	l2 := l.Get(1, 1)
	if l2 != nil {
		t.Errorf("Get on an empty spot should have given a nil locator, got %v", l2)
	}

	o, l2 := l.Put(1, 1, 12, PutAlways)
	if o != nil {
		t.Errorf("Put() on an empty spot should have nil orig value, got %v", o)
	}
	if l2 == nil {
		t.Errorf("Put() on an empty spot should have a non-nil locator, got nil")
	}

	l3 := l.Get(1, 1)
	if l3 != l2 {
		t.Errorf("Get() on the same spot as the previous Put should have given same locator, expected %v got %v", l2, l3)
	}

	l4 := l3.Get(-1, -1)
	if l4 != l {
		t.Errorf("Reciprocal Get() should have given the original locator, expected %v got %v", l, l4)
	}
}

func TestGetWrap(t *testing.T) {
	g := New(2, 2, nil)
	_, l := g.Put(1, 1, 11, PutAlways)
	_, lx := l.Put(1, 0, 12, PutAlways)
	_, ly := l.Put(0, 1, 13, PutAlways)
	lx2 := lx.Get(1, 0)
	if l != lx2 {
		t.Errorf("Get() wrapping around horizontal should have given the original locator, expected %v got %v", l, lx2)
	}
	ly2 := ly.Get(0, 1)
	if l != ly2 {
		t.Errorf("Get() wrapping around vertical should have given the original locator, expected %v got %v", l, ly2)
	}
}

func TestMove(t *testing.T) {
	g := New(3, 3, nil)
	_, l := g.Put(1, 1, 11, PutAlways)
	l2 := g.Get(1, 1)
	if l != l2 {
		t.Errorf("Get() should return same locator as Put(), expected %v got %v", l, l2)
	}

	_, ok := l.Move(1, 0, PutAlways)
	if !ok {
		t.Errorf("Move should have returned ok, got false")
	}

	l3 := g.Get(1, 1)
	if l3 != nil {
		t.Errorf("Move should mean Get returns nil at original spot, got %v", l3)
	}
	l4 := g.Get(2, 1)
	if l4 != l {
		t.Errorf("Get should return original moved locator, expected %v got %v", l, l4)
	}

	_, ok = l.Move(-1, 0, PutAlways)
	if !ok {
		t.Errorf("Move should have returned ok, got false")
	}
	l5 := g.Get(1, 1)
	if l5 != l {
		t.Errorf("Get should return original moved locator, expected %v got %v", l, l5)
	}
}

// 	Replace(n interface{}) Locator
func TestReplace(t *testing.T) {
	g := New(3, 3, nil)
	_, l := g.Put(1, 1, 11, PutAlways)

	if l.Value() != 11 {
		t.Errorf("Value should be 11, got %v", l.Value())
	}

	l2 := l.Replace(12)
	if l2 == nil {
		t.Errorf("Replace should result in non-nil locator")
	}
	if l2 == l {
		t.Errorf("Replace should result in new locator")
	}
	if l.IsValid() {
		t.Errorf("Replace should have invalidated old locator")
	}
	if l2.Value() != 12 {
		t.Errorf("Replcated locator should have value 12, got %v", l2.Value())
	}

	l3 := g.Get(1, 1)
	if l2 != l3 {
		t.Errorf("Get() should return replaced locator, expected %v got %v", l2, l3)
	}
	if l3.Value() != 12 {
		t.Errorf("Replcated locator should have value 12, got %v", l3.Value())
	}
}
