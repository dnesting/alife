package chanbuf

import "testing"

func TestTrigger(t *testing.T) {
	q := Trigger()
	q.Put(1)

	actual, ok := q.Get()
	if actual != nil {
		t.Errorf("trigger get should have produced a nil")
	}
	if !ok {
		t.Errorf("trigger should have produced an ok value")
	}

	q.Done()
	actual, ok = q.Get()
	if ok {
		t.Errorf("should not have gotten ok after Done")
	}
	if len(actual) != 0 {
		t.Errorf("should have gotten empty result, got %v", actual)
	}
}
