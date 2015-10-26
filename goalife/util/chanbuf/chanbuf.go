// Package chanbuf contains functions for queueing, buffering and rate-limiting channels.
//
// Queues accumulate one or more values from a source channel and make those values available
// when the queue's Next() method is invoked.
//
// Buffers provide a channel interface to a queue.
package chanbuf

import "sync"

type QueueWriter interface {
	// Add adds an item to the queue.
	Add(data interface{})
	// Done signals that no more items will be added to the queue.
	Done()
}

type QueueReader interface {
	// Next retrieves values in the queue and will block until values are available.
	Next() []interface{}
}

// A Queue is a queue that emits an aggregation of values from its inputs.
type Queue interface {
	QueueReader
	QueueWriter
}

// queue is the underlying implementation for the queue.
type queue struct {
	cond *sync.Cond
	data []interface{}
	stop bool
	fn   func(existing []interface{}, next interface{}) []interface{}
}

func initQueue() queue {
	return queue{
		cond: sync.NewCond(&sync.Mutex{}),
	}
}

func (q *queue) checkNotStopped() {
	if q.stop {
		panic("Add() called on queue after Done()")
	}
}

func (q *queue) Next() []interface{} {
	q.c.L.Lock()
	defer q.c.L.Unlock()
	for q.u == nil && !q.stop {
		q.c.Wait()
	}
	u := q.u
	q.u = nil
	return u
}

func (q *queue) Done() {
	q.c.L.Lock()
	defer q.c.L.Unlock()
	q.stop = true
	q.c.Signal()
}

func (q *queue) Add(data interface{}) {
	q.queue.data = q.fn(q.queue.data, data)
}

// QueueFilter is a func that decides how the queue returned from
// NewQueue will aggregate its values.  It is passed the existing
// values (which may be nil on the first invocation) and the next
// value received, and is expected to return the new aggregation.
type QueueFilter func(existing []interface{}, next interface{}) []interface{}

func newQueue(filter QueueFilter) *queue {
	return &queue{
		cond: sync.NewCond(&sync.Mutex{}),
		fn:   filter,
	}
}

// FromChan constructs a Queue receiving from the given channel
// and aggregating according to the provided filter.  The queue will
// stop yielding values from Next when ch is closed.
func FromChan(ch <-chan interface{}, filter QueueFilter) QueueReader {
	q := newQueue(fn)
	go chanToQueue(ch, q)
	return q
}

func chanToQueue(source <-chan interface{}, q QueueWriter) {
	for u := range source {
		q.Add(u)
	}
	q.Done()
}

// KeepFirst is a QueueFilter that results in the queue holding
// only the first value obtained from the source when the queue is
// empty.  Additional values that arrive from source when the queue
// is full are discarded and the source should never block.
func KeepFirst(existing []interface{}, next interface{}) []interface{} {
	if existing == nil {
		return []interface{}{next}
	} else {
		return existing
	}
}

// KeepLast is a QueueFilter that results in the queue holding
// only the last value obtained from the source.  Other values that
// arrive from the source before the most recent value are discarded
// and the source should never block.
func KeepLast(existing []interface{}, next interface{}) []interface{} {
	if existing == nil {
		return []interface{}{next}
	} else {
		existing[0] = next
		return existing
	}
}

// KeepAll is a QueueFilter that results in the queue retaining
// everything received from source.  There is no memory bound on
// such a queue.
func KeepAll(existing []interface{}, next interface{}) []interface{} {
	return append(existing, next)
}

// Buffer reads from q and sends the received data to sink. When the
// queue reports it is exhausted, sink will be closed.
func Buffer(q QueueReader, sink chan<- interface{}) {
	for {
		data, ok := q.Next()
		if ok {
			sink <- data
		} else {
			break
		}
	}
	close(sink)
}

// Buffered exposes q as a channel, running Buffer concurrently.
func Buffered(q QueueReader) <-chan interface{} {
	sink := make(chan interface{}, 0)
	go Buffer(q, sink)
	return sink
}
