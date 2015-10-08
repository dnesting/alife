// Package census implements a method for tracking types of things.
package census

import "fmt"

type Key interface {
	Hash() uint64
}

// Population describes the presence of a species in a world.
type Population struct {
	Key Key

	Count int         // number of items in this population currently
	First interface{} // first time the population was seen in the world
	Last  interface{} // last time the population was seen in this world, undefined when Count>0
}

func (c *Population) String() string {
	return fmt.Sprintf("[population %d count=%d (%d-%d)]", c.Key, c.Count, c.First, c.Last)
}

// Census is a type that is used to track changes in a world, grouped by a key.
type Census interface {
	Get(key Key) (Population, bool)
	Add(when interface{}, key Key) Population
	Remove(when interface{}, key Key) Population
	Count() int
	CountAllTime() int
	Distinct() int
	DistinctAllTime() int
}
