// Package census implements a method for tracking genomes.
//
// A Genome consists of (a) an identifying uint32 hash and (b) a []string
// describing the full contents of the genome.  The meaning and derivation
// of these two items is implementation-dependent.
package census

import "fmt"

// Genomer is anything that can provide a genome.
type Genomer interface {
	Genome() Genome
}

// Genome is a type that describes what makes an organism "genetically distinct". It
// consists of a uint32 hash and a []string describing the genome.  The meaning of
// these is implementation-defined.
type Genome interface {
	Hash() uint32
	Code() []string
}

// Cohort describes the presence of a species in a world.
type Cohort struct {
	Genome Genome
	Count  int   // population of this cohort
	First  int64 // first time the genome was seen in the world
	Last   int64 // last time the genome was seen in this world
}

func (c *Cohort) String() string {
	return fmt.Sprintf("[cohort %d count=%d (%d-%d)]", c.Genome.Hash(), c.Count, c.First, c.Last)
}

// OnChangeCallback is a type used to communicate changes to the Census.
type OnChangeCallback func(b Census, c *Cohort, added bool)

// Census is a type that is used to track changes in a world, grouped by Genome.
type Census interface {
	Add(when int64, genome Genome) *Cohort
	Remove(when int64, genome Genome) *Cohort
	Count() int
	CountAllTime() int
	Distinct() int
	DistinctAllTime() int
	OnChange(fn OnChangeCallback)
}
