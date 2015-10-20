package chanbuf

import "time"

// Tick buffers messages from source and provides them to the returned channel at each
// interval.  If always is true, sends nil values at intervals where nothing was buffered.
// Closes the returned sink channel when the source channel is closed.
func Tick(source <-chan interface{}, interval time.Duration, always bool) <-chan []interface{} {
	sink := make(chan []interface{}, buf)
	go tickerLoop(sink, source, interval, always)
	return sink
}

func tickerLoop(sink chan<- []interface{}, source <-chan interface{}, interval time.Duration, always bool) {
	ticker := time.NewTicker(interval)

	var pending []interface{}
	for {
		select {
		case data, ok := <-source:
			if !ok {
				close(sink)
				ticker.Stop()
				return
			}
			pending = append(pending, data)
		case <-ticker.C:
			if pending != nil || always {
				sink <- pending
				pending = nil
			}
		}
	}
}

// RateLimited buffers messages so that no two messages are delivered less than min apart.
// Messages are generally sent when they are received, unless the min duration hasn't passed
// yet, in which case the message will be held and delivered (with any others buffered in the
// same time window) when min has elapsed.  This function is a shortcut for running RateLimit
// concurrently on a newly created and returned chan.
func RateLimited(source <-chan interface{}, min time.Duration) <-chan []interface{} {
	sink := make(chan []interface{}, 0)
	go RateLimit(sink, source, min)
	return sink
}

// RateLimit buffers messages so that no two messages are delivered less than min apart.
// Messages are generally sent when they are received, unless the min duration hasn't passed
// yet, in which case the message will be held and delivered (with any others buffered in the
// same time window) when min has elapsed.  This function will return when source is closed
// and any pending messages have been delivered.
func RateLimit(sink chan<- []interface{}, source <-chan interface{}, min time.Duration) {
	var timer *time.Timer
	var timeCh <-chan time.Time
	var pending []Update
	var waiting bool
	var due time.Time

	// Sends anything pending to sink
	doSend := func(now time.Time) {
		if waiting {
			due = now.Add(min)
			sink <- pending
			pending = nil
			waiting = false
		}
	}

	// Sets a timer so we can delay sending anything pending.
	setTimer := func(d time.Duration) {
		if timer == nil {
			timer = time.NewTimer(d)
			timeCh = timer.C
		} else {
			timer.Reset(d)
		}
	}

	// Read from source or from any pending timer until both are exhausted.
	for source != nil || waiting {
		select {
		case data, ok := <-source:
			if ok {
				pending = append(pending, data)
				waiting = true
				now := time.Now()
				if due.Before(now) {
					doSend(now)
				} else {
					setTimer(due.Sub(now))
				}
			} else {
				// No more data, so ensure this select case is never reached again by setting
				// the channel to nil.
				source = nil
			}
		case <-timeCh:
			doSend(time.Now())
		}
	}
	close(sink)
}
