package census

import "fmt"
import "testing"

type fakeKeyType struct {
	V     int
	Other int
}

func fakeKey(hash int) fakeKeyType {
	return fakeKeyType{hash, 0}
}

func (fk fakeKeyType) Hash() uint64 {
	return uint64(fk.V)
}

func TestEmpty(t *testing.T) {
	var c MemCensus
	_, ok := c.Get(fakeKey(1))
	if ok {
		t.Error("Get with empty census should not return ok")
	}
	if c.Count() != 0 {
		t.Errorf("Count on empty census should be 0, got %d", c.Count())
	}
	if c.CountAllTime() != 0 {
		t.Errorf("CountAllTime on empty census should be 0, got %d", c.CountAllTime())
	}
	if c.Distinct() != 0 {
		t.Errorf("Distinct on empty census should be 0, got %d", c.Distinct())
	}
	if c.DistinctAllTime() != 0 {
		t.Errorf("DistinctAllTime on empty census should be 0, got %d", c.DistinctAllTime())
	}
}

func TestAddRemove(t *testing.T) {
	var c MemCensus

	key := fakeKey(10)
	key.Other = 42
	p := c.Add(1, key)
	if p.Key != key {
		t.Errorf("Population.Key does not match added key, got %v expected %v", p.Key, key)
	}
	if p.Count != 1 {
		t.Errorf("Population count should be 1, got %v", p.Count)
	}
	if p.First != 1 {
		t.Errorf("Population first sighting should be 1, got %v", p.First)
	}
	if p.Last != nil {
		t.Errorf("New population should have nil Last, got %v", p.Last)
	}

	if c.Count() != 1 {
		t.Errorf("Count on census should be 1, got %d", c.Count())
	}
	if c.CountAllTime() != 1 {
		t.Errorf("CountAllTime on census should be 1, got %d", c.CountAllTime())
	}
	if c.Distinct() != 1 {
		t.Errorf("Distinct on census should be 1, got %d", c.Distinct())
	}
	if c.DistinctAllTime() != 1 {
		t.Errorf("DistinctAllTime on census should be 1, got %d", c.DistinctAllTime())
	}

	p = c.Remove(2, fakeKey(10))
	if p.Key != key {
		t.Errorf("Population.Key does not match added key, got %v expected %v", p.Key, key)
	}
	if p.Count != 0 {
		t.Errorf("Population count should be 0, got %v", p.Count)
	}
	if p.First != 1 {
		t.Errorf("Population first sighting should be 1, got %v", p.First)
	}
	if p.Last != 2 {
		t.Errorf("Population last sighting should be 2, got %v", p.Last)
	}

	if c.Count() != 0 {
		t.Errorf("Count on census should be 0, got %d", c.Count())
	}
	if c.CountAllTime() != 1 {
		t.Errorf("CountAllTime on census should be 1, got %d", c.CountAllTime())
	}
	if c.Distinct() != 0 {
		t.Errorf("Distinct on census should be 1, got %d", c.Distinct())
	}
	if c.DistinctAllTime() != 1 {
		t.Errorf("DistinctAllTime on census should be 1, got %d", c.DistinctAllTime())
	}
}

func TestMultiple(t *testing.T) {
	var c MemCensus
	c.Add(1, fakeKey(10))
	c.Add(2, fakeKey(20))
	c.Add(3, fakeKey(20))
	c.Add(4, fakeKey(10))
	c.Add(5, fakeKey(30))
	c.Remove(6, fakeKey(30))
	c.Remove(7, fakeKey(10))

	if c.Count() != 3 {
		t.Errorf("Count should be 3, got %d", c.Count())
	}
	if c.CountAllTime() != 5 {
		t.Errorf("CountAllTime should be 5, got %d", c.CountAllTime())
	}
	if c.Distinct() != 2 {
		t.Errorf("Distinct should be 2, got %d", c.Distinct())
	}
	if c.DistinctAllTime() != 3 {
		t.Errorf("DistinctAllTime should be 3, got %d", c.DistinctAllTime())
	}

	p, _ := c.Get(fakeKey(20))
	if p.Count != 2 {
		t.Errorf("Population count should be 2, got %d", p.Count)
	}
}

type IntKey int

func (k IntKey) Hash() uint64 {
	return uint64(k)
}

func ExampleMemCensus() {
	var c MemCensus
	c.Add(1, IntKey(10))
	c.Add(2, IntKey(10))
	c.Add(3, IntKey(20))
	c.Add(4, IntKey(30))
	c.Remove(5, IntKey(20))

	fmt.Printf("%d added, %d still there, %d distinct\n", c.CountAllTime(), c.Count(), c.Distinct())
	// Output: 4 added, 3 still there, 2 distinct
}
