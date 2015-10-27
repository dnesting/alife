package chanbuf

import "container/ring"
import "sync"

type ringQueue struct {
	cond   *sync.Cond
	read   *ring.Ring
	insert *ring.Ring
	done   bool
}

// Ring creates a Queue that only retains the last size elements.
func Ring(size int) Queue {
	q := &ringQueue{
		cond:   sync.NewCond(&sync.Mutex{}),
		insert: ring.New(size),
	}
	q.read = q.insert
	return q
}

func (q *ringQueue) Put(value interface{}) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	if q.done {
		panic("Put called after Done")
	}
	q.insert.Value = value
	if q.read == nil {
		q.read = q.insert
	} else if q.read == q.insert {
		q.read = q.read.Next()
	}
	q.insert = q.insert.Next()
	q.cond.Signal()
}

func (q *ringQueue) Done() {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	q.done = true
	q.cond.Signal()
}

func (q *ringQueue) Get() ([]interface{}, bool) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	for q.read == nil && !q.done {
		q.cond.Wait()
	}

	if q.read == nil { // implies q.done
		return nil, false
	}

	var values []interface{}
	for {
		values = append(values, q.read.Value)
		q.read = q.read.Next()
		if q.read == q.insert {
			break
		}
	}
	q.read = nil
	return values, true
}
