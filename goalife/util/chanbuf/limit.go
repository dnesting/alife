package chanbuf

import "sync"

type limitQueue struct {
	cond   *sync.Cond
	limit  int
	values []interface{}
	done   bool
}

// Limit creates a Queue that only retains the first size elements.
// A size of 0 will result in all values being discarded.
func Limit(size int) Queue {
	return &limitQueue{
		cond:  sync.NewCond(&sync.Mutex{}),
		limit: size,
	}
}

// Unlimited creates a Queue with an unbounded queue size.
func Unlimited() Queue {
	return Limit(-1)
}

// Discard creates a Queue that drops all items added to it.  Get
// will block without returning any values until Done is called.
// If you would like to drop values while unblocking Get when a
// value arrives, see Trigger.
func Discard() Queue {
	return Limit(0)
}

func (q *limitQueue) Put(value interface{}) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	if q.done {
		panic("Put called after Done")
	}

	if q.limit < 0 || len(q.values) < q.limit {
		q.values = append(q.values, value)
		q.cond.Signal()
	}
}

func (q *limitQueue) Done() {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	q.done = true
	q.cond.Signal()
}

func (q *limitQueue) Get() ([]interface{}, bool) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	for len(q.values) == 0 && !q.done {
		q.cond.Wait()
	}

	if len(q.values) == 0 {
		return nil, false
	}

	values := q.values
	q.values = nil
	return values, true
}
