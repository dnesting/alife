// Package census implements a method for tracking populations grouped by a key.
package census

import "fmt"

// Key is a way for the caller to group similar types of things.  Typically the
// caller might make this some identifying characteristic of the things, and generate
// a hash that we can use to distinguish in a standard way.
type Key interface {
	Hash() uint64
}

// Population describes the presence of a group of things with the same key.
// First and Last record the "when" event for the corresponding Add/Remove events,
// the type of which is implementation- or caller-defined.
type Population struct {
	Key Key

	Count int         // number of items in this population currently
	First interface{} // first time the population was seen
	Last  interface{} // last time the population was seen
}

func (c *Population) String() string {
	return fmt.Sprintf("[population %v count=%d (%d-%d)]", c.Key, c.Count, c.First, c.Last)
}

// A Census serves as a way of counting the appearance or removal of a thing.
// During its life, it will typically keep a running count of the number of things
// with the same key hash.
type Census interface {
	Get(key Key) (Population, bool)
	Add(when interface{}, key Key) Population
	Remove(when interface{}, key Key) Population
	Count() int
	CountAllTime() int
	Distinct() int
	DistinctAllTime() int
}
