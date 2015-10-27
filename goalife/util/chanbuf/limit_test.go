package chanbuf

import "reflect"
import "testing"

func TestLimit(t *testing.T) {
	var tests = []struct {
		toPut    []int
		expected []interface{}
	}{
		{[]int{1, 2, 3, 4}, []interface{}{1, 2, 3}},
		{[]int{5, 6, 7, 8}, []interface{}{5, 6, 7}},
	}

	q := Limit(3)
	for _, test := range tests {
		for _, v := range test.toPut {
			q.Put(v)
		}
		actual, ok := q.Get()

		if !ok {
			t.Errorf("expected ok result from Get, got false")
		}
		if !reflect.DeepEqual(test.expected, actual) {
			t.Errorf("expected %v got %v", test.expected, actual)
		}
	}

	q.Done()
	actual, ok := q.Get()
	if ok {
		t.Errorf("should not have gotten ok after Done")
	}
	if len(actual) != 0 {
		t.Errorf("should have gotten empty result, got %v", actual)
	}
}
