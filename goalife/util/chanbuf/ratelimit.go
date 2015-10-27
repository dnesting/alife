package chanbuf

import "sync"
import "time"

// deps is where we set up our dependencies so that we can replace them during testing.
var deps = struct {
	newTicker func(time.Duration) *time.Ticker
	sleep     func(time.Duration)
}{
	time.NewTicker,
	time.Sleep,
}

// Tick delivers items from source only every interval.  If always
// is true, sends nil values at intervals where nothing was buffered.
// Closes the returned sink channel when the source channel is
// closed.
func Tick(source QueueGetter, interval time.Duration, always bool) <-chan []interface{} {
	sink := make(chan []interface{})
	go tickerLoop(sink, source, interval, always)
	return sink
}

func tickerLoop(sink chan<- []interface{}, source QueueGetter, interval time.Duration, always bool) {
	ticker := deps.newTicker(interval)

	pending := struct {
		sync.Mutex
		data     []interface{} // data retrieved from source, or nil
		received bool          // data was something actually received from source and not initial nil
		proceed  bool          // last receive operation succeeded, so keep going
	}{
		proceed: true,
	}

	// Fetches a single item from source and stores it in pending.
	fetch := func() {
		data, ok := source.Get()

		pending.Lock()
		defer pending.Unlock()

		pending.data = data
		pending.received = ok
		pending.proceed = ok
	}

	// Returns the last item received by fetch.
	get := func() ([]interface{}, bool, bool) {
		pending.Lock()
		defer pending.Unlock()

		data := pending.data
		received := pending.received
		pending.data = nil
		pending.received = false
		// Keep pending.ok set to true so we know we're not out of values yet.
		return data, received, pending.proceed
	}

	done := make(chan struct{})

	// Start monitoring the ticker and looking for data to sink concurrently.
	go func() {
		for _ = range ticker.C {
			value, received, proceed := get()
			if !proceed {
				break
			}
			if received {
				go fetch() // get another concurrently
				sink <- value
			} else if always {
				sink <- nil
			}
		}
		ticker.Stop()
		close(done)
	}()

	// Do our initial fetch in the parent goroutine to maximize the chance
	// that if something is available it'll be ready for the above goroutine
	// during its first loop.
	fetch()

	// Wait for the goroutine to exit.
	<-done
	close(sink)
}

// RateLimited reads from source no more often than min and sends the result to
// the returned chan.  This function is a shortcut for running RateLimit
// concurrently on a newly created and returned chan.
func RateLimited(source QueueGetter, min time.Duration) <-chan []interface{} {
	sink := make(chan []interface{}, 0)
	go RateLimit(sink, source, min)
	return sink
}

// RateLimit reads from source no more often than min and delivers the result to sink.
// This function will return when source stops providing values.
func RateLimit(sink chan<- []interface{}, source QueueGetter, min time.Duration) {
	for {
		if data, ok := source.Get(); ok {
			sink <- data
			deps.sleep(min)
		} else {
			break
		}
	}
	close(sink)
}
