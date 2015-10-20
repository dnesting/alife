// Package chanbuf contains functions for queueing, buffering and rate-limiting channels.
//
// Queues accumulate one or more values from a source channel and make those values available
// when the queue's Next() method is invoked.
//
// Buffers provide a channel interface to a queue.
package chanbuf

type Accumulator interface {
	// Add adds an item to the queue.
	Add(data interface{})
	// Done signals that no more items will be added to the queue.
	Done()
}

// SingleValue yields a single value from Next().
type SingleValue interface {
	Next() interface{}
}

// A SingleQueue is a queue that only emits a single value from its queue.
type SingleQueue interface {
	Accumulator
	SingleValue
}

// MultiValue yields multiple values from Next().
type MultiValue interface {
	Next() []interface{}
}

// A MultiQueue is a queue that emits an aggregation of values from its queue.
type MultiQueue interface {
	Accumulator
	MultiValue
}

// queue is the underlying implementation for the queues.  We use the interface{} type
// both in the single and multi versions of the queue.
type queue struct {
	cond *sync.Cond
	data interface{}
	stop bool
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

func (q *queue) Next() interface{} {
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

// singleQueue is a queue implementation that selectively chooses a single
// value to hold in the queue.
type singleQueue struct {
	queue
	fn func(existing, next interface{}) interface{}
}

func (q *singleQueue) Add(data interface{}) {
	q.queue.data = q.fn(q.queue.data, data)
}

// queueAll is a queue implementation that holds a slice of values accumulated
// through Add calls.
type queueAll struct {
	queue queue
}

func (q *queueAll) Next() []interface{} {
	if value := q.queue.Next(); value != nil {
		return value.([]interface{})
	}
	return nil
}

func (q *queueAll) Add(data interface{}) {
	q.queue.c.L.Lock()
	defer q.queue.c.L.Unlock()
	q.queue.data = append(q.queue.data, data)
	q.c.Signal()
}

func (q *queueAll) Done() {
	q.queue.Done()
}

func queueSingle(ch <-chan interface{}, filter func(existing, next interface{}) interface{}) SingleQueue {
	q := singleQueue{initQueue(), filter}
	go chanToQueue(ch, q)
	return q
}

// Chan converts a readable channel of any type to a chan interface{}. This may simplify
// calling some of the functions in this package.
func Chan(ch interface{}) chan interface{} {
	out := make(chan interface{})
	chValue := reflect.ValueOf(ch)

	go func() {
		defer close(out)
		for {
			value, ok := chValue.Recv()
			if !ok {
				return
			}
			out <- value.Interface()
		}
	}()
	return out
}

// QueueFirst creates a queue that holds a single value: the first value obtained from source whenever the
// queue is empty. Additional values that arrive from source when the queue is full are discarded and source
// should never block.
func QueueFirst(source <-chan interface{}) SingleQueue {
	return queueSingle(source, func(first, _ interface{}) interface{} { return first })
}

// QueueLast creates a queue that holds a single value: the most recent value obtained from source. Values
// that arrived between a call to Next() and the most recent value received from source are discarded.
func QueueLast(source <-chan interface{}) SingleQueue {
	return queueSingle(source, func(_, last interface{}) interface{} { return last })
}

func chanToQueue(source <-chan interface{}, q SingleQueue) {
	for u := range source {
		q.Add(u)
	}
	q.Done()
}

// QueueAll creates a queue that holds all values received from source.  These values are then returned
// as a slice by calling Next().  There is no memory bound on this queue.
func QueueAll(source <-chan interface{}) MultiQueue {
	q := queueAll{initQueue()}
	go chanToQueue(source, q)
	return q
}

// BufferFirst creates a buffer between channels, allowing the source channel to remain unblocked
// even if the sink channel is not ready.  See QueueFirst for more information.
func BufferFirst(source <-chan interface{}, sink chan<- interface{}) {
	bufferSingle(QueueFirst(source), sink)
}

// BufferLast creates a buffer between channels, allowing the source channel to remain unblocked
// even if the sink channel is not ready.  See QueueLast for more information.
func BufferLast(source <-chan interface{}, sink chan<- interface{}) {
	bufferSingle(QueueLast(source), sink)
}

func bufferSingle(q SingleQueue, sink chan<- interface{}) {
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

// BufferAll creates a buffer between channels, allowing the source channel to remain unblocked
// even if the sink channel is not ready.  See QueueAll for more information.
func BufferAll(source <-chan interface{}, sink chan<- []interface{}) {
	q := QueueAll(source)
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

// BufferedFirst runs BufferFirst in the background on a newly created and returned channel.
func BufferedFirst(source <-chan interface{}) <-chan interface{} {
	sink := make(chan interface{}, 0)
	go BufferFirst(source, sink)
	return sink
}

// BufferedLast runs BufferLast in the background on a newly created and returned channel.
func BufferedLast(source <-chan interface{}) <-chan interface{} {
	sink := make(chan interface{}, 0)
	go BufferLast(source, sink)
	return sink
}

// BufferedAll runs BufferAll in the background on a newly created and returned channel.
func BufferedAll(source <-chan interface{}) <-chan []interface{} {
	sink := make(chan []interface{}, 0)
	go BufferAll(source, sink)
	return sink
}
