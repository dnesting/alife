// Package chanbuf contains functions for queueing and rate-limiting a supply of data in a
// way that does not block the producer.
//
// Queues accumulate one or more values from a source and make those values available
// when the queue's Get() method is invoked.
//
// Buffers provide a channel interface to a Queue.
package chanbuf

// QueuePutter is the Put/Done side of the Queue interface, used by components
// that add items onto a queue.
type QueuePutter interface {
	// Put adds an item to the queue. This is guaranteed not to block.
	Put(data interface{})
	// Done signals that no more items will be added to the queue. It is illegal
	// to call Put after calling Done.
	Done()
}

// QueueGetter is the Get side of the Queue interface, used by components that
// simply retrieve values from a queue.
type QueueGetter interface {
	// Get retrieves values in the queue and will block until values are available.
	// If no more data is available to be consumed, returns (nil, false).
	Get() (data []interface{}, ok bool)
}

// Queue is a queue that emits an aggregation of values from its inputs.  QueueGetter
// and QueuePutter are separated out to clearly differentiate between consumer and
// producer roles where needed.
type Queue interface {
	QueueGetter
	QueuePutter
}

// Feed calls q.Put on every value received from source, and invokes q.Done when
// the channel is closed.
func Feed(q QueuePutter, source <-chan interface{}) {
	for v := range source {
		q.Put(v)
	}
	q.Done()
}

// Buffer reads from q and sends the received data to sink. When the
// queue reports it is exhausted, sink will be closed.
func Buffer(q QueueGetter, sink chan<- interface{}) {
	for {
		data, ok := q.Get()
		if ok {
			sink <- data
		} else {
			break
		}
	}
	close(sink)
}

// Buffered exposes q as a channel, running Buffer concurrently.
func Buffered(q QueueGetter) <-chan interface{} {
	sink := make(chan interface{}, 0)
	go Buffer(q, sink)
	return sink
}
