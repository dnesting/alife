package chanbuf

import "sync"

type triggerQueue struct {
	cond      *sync.Cond
	triggered bool
	done      bool
}

// Trigger creates a Queue that retains no elements and simply
// makes a single nil available to Get if Put was called any
// number of times since the last Get.
func Trigger() Queue {
	return &triggerQueue{
		cond: sync.NewCond(&sync.Mutex{}),
	}
}

func (q *triggerQueue) Put(value interface{}) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	if q.done {
		panic("Put called after Done")
	}

	q.triggered = true
	q.cond.Signal()
}

func (q *triggerQueue) Done() {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	q.done = true
	q.cond.Signal()
}

func (q *triggerQueue) Get() ([]interface{}, bool) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	for !q.triggered && !q.done {
		q.cond.Wait()
	}

	if !q.triggered {
		return nil, false
	}

	q.triggered = false
	return nil, true
}
