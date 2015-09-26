// Package stats contains some types for tracking statistics
package stats

import "container/ring"
import "sync"
import "time"

// For faking the clock during tests
type clock interface {
	Now() time.Time
}

type realClock struct{}

func (r realClock) Now() time.Time { return time.Now() }

var clk clock = realClock{}

// CumulativeFloat64 is a type that accumulates timeseries values and gives you some aggregation.
type CumulativeFloat64 interface {
	Add(v float64)
	Value() float64
	Valid() bool
}

// entry is a single data point in a timeseries.
type entry struct {
	V float64
	T time.Time
}

// ringStat accumulates values into a container.Ring, filtering old values based upon
// the keep func.
type ringStat struct {
	mu sync.RWMutex
	r  *ring.Ring // always points to the earliest node added; r.Prev() is latest
}

// Valid is true when Value() is expected to provide meaningful results.  Some
// cumulative metrics may be invalid, for instance, if they haven't accumulated
// a single data point yet.
func (s *ringStat) Valid() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.r != nil && s.r.Len() > 0
}

// Add accumulates a new value at the present time.
func (s *ringStat) Add(v float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.addLocked(v, clk.Now())
}

// addLocked accumulates a new value at the specific time.
func (s *ringStat) addLocked(v float64, t time.Time) {
	e := entry{v, t}
	n := ring.New(1)
	n.Value = e

	if s.r == nil {
		s.r = n
	} else {
		// s.r.Prev() is always the latest node added, so append to that
		s.r.Prev().Link(n)
	}
}

// prune starts from the oldest value (s.r) and removes elements from the
// ring when they do not satisfy s.keep.
func (a *MovingAvg) pruneLocked() {
	del := 0
	for i := a.r.r; i != a.r.r.Prev(); i = i.Next() {
		e := i.Value.(entry)
		if clk.Now().Sub(e.T) < a.Duration {
			// assume all elements after this one are at a later time
			break
		} else {
			del += 1
		}
	}
	if del == a.r.r.Len() {
		a.r.r = nil
	} else if del > 0 {
		p := a.r.r.Prev()
		p.Unlink(del)
		a.r.r = p.Next()
	}
}

// MovingAvg represents an accumulation of values that will be pruned with the
// given duration and whose aggregation is an average.
type MovingAvg struct {
	Duration time.Duration
	r        ringStat
	mu       sync.Mutex
}

func (a *MovingAvg) Add(v float64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.r.Add(v)
	a.pruneLocked()
}

// Value retrieves the current metric value.  The value retrieved is undefined
// if Valid() returns false.
func (a *MovingAvg) Value() float64 {
	a.mu.Lock()
	defer a.mu.Unlock()

	num := 0
	avg := 0.0
	a.pruneLocked()
	if !a.r.Valid() {
		return avg
	}
	a.r.r.Do(func(i interface{}) {
		e := i.(entry)
		num += 1
		avg = (e.V + float64(num-1)*avg) / float64(num)
	})
	return avg
}
