package chanbuf

import "time"

// Tick buffers messages from source and provides them to the returned channel at each
// interval.  If always is true, sends nil values at intervals where nothing was buffered.
// Closes the returned sink channel when the source channel is closed.
func Tick(source QueueReader, interval time.Duration, always bool) <-chan []interface{} {
	sink := make(chan []interface{})
	go tickerLoop(sink, source, interval, always)
	return sink
}

func tickerLoop(sink chan<- []interface{}, source QueueReader, interval time.Duration, always bool) {
	ticker := time.NewTicker(interval)

	pending := struct {
		sync.Mutex
		data     []interface{} // data retrieved from source, or nil
		received bool          // data was something actually received from source and not initial nil
		proceed  bool          // last receive operation succeeded, so keep going
	}{
		proceed: true,
	}

	// Fetches a single item from source and stores it in pending.
	recv := func() {
		data, ok = source.Next()

		pending.Lock()
		defer pending.Unlock()

		pending.data = data
		pending.received = ok
		pending.proceed = ok
	}

	// Fetches the last item received by recv.
	get := func() ([]interface{}, bool, bool) {
		pending.Lock()
		defer pending.Unlock()

		data = pending.data
		received = pending.received
		pending.data = nil
		pending.received = false
		// Keep pending.ok set to true so we know we're not out of values yet.
		return data, received, pending.proceed
	}

	// Begin by fetching the first value from source.
	go recv()

	for _ := range ticker.C {
		value, received, proceed := get()
		if !proceed {
			break
		}
		if received {
			go recv() // get another concurrently
			sink <- value
		} else if always {
			sink <- nil
		}
	}
	ticker.Stop()
	close(sink)
}

// RateLimited reads from source no more often than min and sends the result to
// the returned chan.  This function is a shortcut for running RateLimit
// concurrently on a newly created and returned chan.
func RateLimited(source QueueReader, min time.Duration) <-chan []interface{} {
	sink := make(chan []interface{}, 0)
	go RateLimit(sink, source, min)
	return sink
}

// RateLimit reads from source no more often than min and delivers the result to sink.
// This function will return when source stops providing values.
func RateLimit(sink chan<- []interface{}, source QueueReader, min time.Duration) {
	for {
		if data, ok := source.Next(); ok {
			sink <- data
			time.Sleep(min)
		} else {
			break
		}
	}
	close(sink)
}
