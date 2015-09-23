// Package stats contains some types for tracking statistics
package stats

import "sync/atomic"

// CumulativeInt64 is a type that accumulates timeseries values and gives you some aggregation.
type CumulativeInt64 interface {
	Add(v int64)
	Value() int64
	Valid() bool
}

// Counter is a type that simply accumulates a static value.  This type is
// concurrency-safe if mutations occur through the provided methods.
type Counter struct {
	V int64
}

func (c *Counter) Add(v int64)  { atomic.AddInt64(&c.V, v) }
func (c *Counter) Value() int64 { return atomic.LoadInt64(&c.V) }
func (c *Counter) Valid() bool  { return true }
