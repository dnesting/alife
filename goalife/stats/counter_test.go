package stats

import "testing"

func TestCounter(t *testing.T) {
	c := Counter{3}
	if !c.Valid() {
		t.Fatal("initial Counter should be valid")
	}
	if c.Value() != 3 {
		t.Errorf("initial Counter value should be 3, got %d\n", c.Value())
	}
	c.Add(2)
	if c.Value() != 5 {
		t.Errorf("Counter 3 + 2 value should be 5, got %d\n", c.Value())
	}
}
