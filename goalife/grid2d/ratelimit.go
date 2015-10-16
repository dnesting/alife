package grid2d

import "time"

func RateLimited(source <-chan []Update, freq time.Duration, buf int) <-chan []Update {
	sink := make(chan []Update, buf)
	go RateLimit(sink, source, freq)
	return sink
}

func RateLimit(sink chan<- []Update, source <-chan []Update, freq time.Duration) {
	var due time.Time
	var timeCh <-chan time.Time

	var pending []Update

	doSend := func(now time.Time) {
		due = now.Add(freq)
		sink <- pending
		pending = nil
		timeCh = nil
	}

	doReceive := func(u []Update) {
		now := time.Now()
		if pending == nil {
			pending = u
		} else {
			pending = append(pending, u...)
		}
		if due.Before(now) {
			doSend(now)
		} else {
			if timeCh == nil {
				timeCh = time.After(due.Sub(now))
			}
		}
	}

	for {
		select {
		case u := <-source:
			if u == nil {
				close(sink)
				return
			}
			doReceive(u)
		case <-timeCh:
			doSend(time.Now())
		}
	}
}

func NotifyAsQueue(source <-chan []Update, style NotifyStyle) NotifyQueue {
	q := NewNotifyQueue(style)
	go QueueForNotify(q, source)
	return q
}

func QueueForNotify(q NotifyQueue, source <-chan []Update) {
	for u := range source {
		q.Add(u)
	}
	q.Done()
}
