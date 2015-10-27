package chanbuf

import "reflect"
import "sync/atomic"
import "testing"
import "time"

func fakeTicker() (chan<- time.Time, func()) {
	ticker := make(chan time.Time)
	orig := deps.newTicker
	deps.newTicker = func(d time.Duration) *time.Ticker {
		return &time.Ticker{C: ticker}
	}
	return ticker, func() {
		deps.newTicker = orig
	}
}

func TestTick(t *testing.T) {
	ticker, restore := fakeTicker()
	defer restore()

	q := Unlimited()
	q.Put(1)
	q.Put(2)
	q.Put(3)

	ch := Tick(q, 3*time.Second, false)
	ticker <- time.Now()
	got := <-ch
	if len(got) != 3 {
		t.Errorf("expected 3 elements from tick, got %v", got)
	}

	ticker <- time.Now() // empty

	q.Put(4)
	q.Put(5)
	q.Put(6)

	ticker <- time.Now()
	got = <-ch
	if len(got) != 3 {
		t.Errorf("expected 3 elements from tick, got %v", got)
	}

	q.Done()
}

func TestTickAlways(t *testing.T) {
	ticker, restore := fakeTicker()
	defer restore()

	q := Unlimited()
	ch := Tick(q, 3*time.Second, true)

	ticker <- time.Now()
	got := <-ch
	if got != nil {
		t.Errorf("got expected nil, got %v", got)
	}

	ticker <- time.Now()
	got = <-ch
	if got != nil {
		t.Errorf("got expected nil, got %v", got)
	}

	q.Put(1)
	q.Put(2)
	q.Put(3)
	ticker <- time.Now()
	got = <-ch
	if reflect.DeepEqual([]interface{}{1}, got) {
		t.Errorf("expected first element from tick, got %v", got)
	}

	ticker <- time.Now()
	got = <-ch
	if reflect.DeepEqual([]interface{}{2, 3}, got) {
		t.Errorf("expected remaining elements from tick, got %v", got)
	}
	q.Done()
}

func TestRateLimit(t *testing.T) {
	defer func(orig func(d time.Duration)) { deps.sleep = orig }(deps.sleep)
	var slept int32
	deps.sleep = func(d time.Duration) {
		atomic.AddInt32(&slept, int32(d.Seconds()))
	}
	q := Unlimited()

	q.Put(1)
	ch := RateLimited(q, 3*time.Second)
	if atomic.LoadInt32(&slept) != 0 {
		t.Errorf("slept should be 0, got %d", slept)
	}

	got := <-ch
	if !reflect.DeepEqual([]interface{}{1}, got) {
		t.Errorf("expected [1], got %v", got)
	}
	if atomic.LoadInt32(&slept) != 3 {
		t.Errorf("expected to have slept 3, got %v", slept)
	}

	q.Put(2)
	got = <-ch
	if !reflect.DeepEqual([]interface{}{2}, got) {
		t.Errorf("expected [2], got %v", got)
	}
	if atomic.LoadInt32(&slept) != 6 {
		t.Errorf("expected to have slept 6, got %v", slept)
	}

	q.Done()
	got = <-ch
	if got != nil {
		t.Errorf("expected got to be nil after q.Done, got %v", got)
	}
}
