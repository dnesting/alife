package grid2d

import "bytes"
import "encoding/gob"
import "reflect"
import "testing"

func TestBasic(t *testing.T) {
	g := New(5, 10, nil, nil)

	if g.Get(1, 1) != nil {
		t.Errorf("Get(1,1) should result in nil, got %v", g.Get(1, 1))
	}

	w, h := g.Extents()
	if w != 5 {
		t.Errorf("Width should be 5")
	}
	if h != 10 {
		t.Errorf("Height should be 10")
	}
}

func TestPut(t *testing.T) {
	g := New(2, 2, nil, nil)
	orig, loc := g.Put(0, 0, 10, PutAlways)
	if orig != nil {
		t.Errorf("Put() on an empty spot should have nil orig value, got %v", orig)
	}
	if loc == nil {
		t.Errorf("Put() on an empty spot with PutAlways should return a locator, got nil")
	}
	if !loc.IsValid() {
		t.Errorf("Put() on an empty spot should have given a valid locator")
	}
	if loc.Value() != 10 {
		t.Errorf("Get() value should equal what we put in, expected %v got %v", 10, loc.Value())
	}

	loc2 := g.Get(0, 0)
	if loc != loc2 {
		t.Errorf("Get() on the same spot we did a Put() should give the same locator, expected %v got %v", loc, loc2)
	}

	loc3 := g.Get(1, 0)
	if loc3 != nil {
		t.Errorf("Get() on an empty spot should have given a nil locator, got %v", loc3)
	}
}

func TestPut2(t *testing.T) {
	g := New(2, 2, nil, nil)
	_, loc := g.Put(0, 0, 10, PutAlways)
	orig, loc2 := g.Put(0, 0, 15, PutAlways)
	if orig != loc.Value() {
		t.Errorf("Put() on an occupied spot should return its original value, expected %v got %v", loc.Value(), orig)
	}
	if loc2 == nil {
		t.Errorf("Put() on an occupied spot with PutAlways should return a valid locator")
	}
	if !loc2.IsValid() {
		t.Errorf("Put() on top of an existing value should have given a valid locator")
	}
	if loc2.Value() != 15 {
		t.Errorf("Put() on an occupied spot with PutAlways should return the value we put, expected %v got %v", 15, loc.Value())
	}
	if loc.IsValid() {
		t.Errorf("Put() on top of an existing value should have invalidated its locator")
	}
}

func TestPutWhenNil(t *testing.T) {
	g := New(2, 2, nil, nil)
	_, loc := g.Put(0, 0, 10, PutAlways)
	_, loc2 := g.Put(0, 0, 15, PutWhenNil)
	if loc2 != nil {
		t.Errorf("Put() with PutWhenNil on an occupied spot should have returned nil locator, got %v", loc2)
	}
	if !loc.IsValid() {
		t.Errorf("Put() with PutWhenNil on an occupied spot should not have invalidated the existing locator")
	}
	if loc.Value() != 10 {
		t.Errorf("Put() with PutWhenNil on an occupied spot should cause its locator to return its original value, expected %v got %v", 10, loc.Value())
	}
}

func TestAll(t *testing.T) {
	g := New(2, 2, nil, nil)
	all := g.All()
	if len(all) != 0 {
		t.Errorf("All() should have returned an empty slice on an empty grid, got %v", all)
	}

	_, loc := g.Put(0, 0, 10, PutAlways)
	all = g.All()
	expected := []Locator{loc}
	if !reflect.DeepEqual(all, expected) {
		t.Errorf("All() should have returned a one-element slice, expected %v got %v", expected, all)
	}

	_, loc2 := g.Put(1, 1, 10, PutAlways)
	all = g.All()
	expected = []Locator{loc, loc2}
	if !reflect.DeepEqual(all, expected) {
		t.Errorf("All() should have returned a two-element slice, expected %v got %v", expected, all)
	}
}

func TestResize(t *testing.T) {
	g := New(3, 3, nil, nil)
	g.Put(1, 2, 10, PutAlways)
	var ran bool
	g.Resize(2, 2, func(x, y int, o interface{}) {
		ran = true
		if x != 1 || y != 2 || o != 10 {
			t.Errorf("unexpected removed item, expected (%d,%d,%v) got (%d,%d,%v)", 1, 2, 10, x, y, o)
		}
	})
	if !ran {
		t.Errorf("resize should have removed an entity but didn't; world now %v\n", g.All())
	}
}

func TestLocations(t *testing.T) {
	g := New(3, 3, nil, nil)
	var locs []Point
	_, _, count := g.Locations(&locs)
	if len(locs) != 0 {
		t.Errorf("Locations() should have kept an empty slice on an empty grid, got %v", locs)
	}
	if count != 0 {
		t.Errorf("Locations() count should have been 0, got %v", count)
	}

	g.Put(1, 2, 10, PutAlways)
	_, _, count = g.Locations(&locs)
	if len(locs) != 1 {
		t.Errorf("Locations() should return one element when one field is occupied, got %v", len(locs))
	}
	if count != 1 {
		t.Errorf("Locations() count should have been 1, got %v", count)
	}
	expected := Point{1, 2, 10}
	if !reflect.DeepEqual(locs[0], expected) {
		t.Errorf("Locations() should have returned a one-element slice, expected %v got %v", expected, locs[0])
	}

	_, _, count = g.Locations(nil)
	if count != 1 {
		t.Errorf("Locations() count on a nil slice argument should have been 1, got %v", count)
	}
}

func TestGob(t *testing.T) {
	g := New(3, 3, nil, nil)
	g.Put(1, 2, 10, PutAlways)
	var locs []Point
	w, h, _ := g.Locations(&locs)

	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	if err := enc.Encode(g); err != nil {
		t.Errorf("error encoding: %v", err)
	}

	dec := gob.NewDecoder(&b)
	g2 := New(0, 0, nil, nil)
	var locs2 []Point
	w2, h2, _ := g2.Locations(&locs2)
	if w2 != 0 || h2 != 0 || len(locs2) != 0 {
		t.Errorf("zero value for Grid is not zero, got (%d,%d) and locations=%v", w2, h2, locs2)
	}

	if err := dec.Decode(g2); err != nil {
		t.Fatalf("error decoding: %v", err)
	}

	w2, h2, _ = g2.Locations(&locs2)
	if w != w2 {
		t.Errorf("decoded grid has wrong width, expected %d got %d", w, w2)
	}
	if h != h2 {
		t.Errorf("decoded grid has wrong height, expected %d got %d", h, h2)
	}
	if !reflect.DeepEqual(locs, locs2) {
		t.Errorf("decoded grid has wrong contents, expected %v got %v", locs, locs2)
	}
}
