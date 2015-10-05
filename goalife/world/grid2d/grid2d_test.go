package grid2d

import "reflect"
import "testing"

func TestBasic(t *testing.T) {
	g := New(5, 10)

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
	g := New(2, 2)
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
	g := New(2, 2)
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
	g := New(2, 2)
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
	g := New(2, 2)
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

func TestLocations(t *testing.T) {
	g := New(3, 3)
	locs := g.Locations()
	if len(locs) != 0 {
		t.Errorf("Locations() should have returned an empty slice on an empty grid, got %v", locs)
	}

	g.Put(1, 2, 10, PutAlways)
	locs = g.Locations()
	if len(locs) != 1 {
		t.Errorf("Locations() should return one element when one field is occupied, got %v", len(locs))
	}
	expected := Point{1, 2, 10}
	if !reflect.DeepEqual(locs[0], expected) {
		t.Errorf("Locations() should have returned a one-element slice, expected %v got %v", expected, locs[0])
	}
}
