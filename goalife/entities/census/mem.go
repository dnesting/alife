package census

import "sync"

// MemCensus implements a Census entirely in-memory.
type MemCensus struct {
	mu          sync.RWMutex
	Seen        map[uint32]*Cohort
	count       int
	countAll    int
	distinct    int
	distinctAll int
	onChange    OnChangeCallback
	last        *Cohort
}

// NewMemCensus creates a new in-memory Census.
func NewMemCensus() *MemCensus {
	return &MemCensus{
		Seen: make(map[uint32]*Cohort),
	}
}

// Add indicates an instance of the given genome was added to the world.
func (b *MemCensus) Add(when int64, genome Genome) *Cohort {
	var c *Cohort
	func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		k := genome.Hash()
		var ok bool
		c, ok = b.Seen[k]
		if !ok {
			c = &Cohort{
				Genome: genome,
				First:  when,
			}
			b.Seen[k] = c
			b.distinct += 1
			b.distinctAll += 1
		}
		c.Count += 1
		b.count += 1
		b.countAll += 1
	}()
	if b.onChange != nil {
		b.onChange(b, c, true)
	}
	return c
}

// Remove indicates an instance of the given genome was removed from the world.
func (b *MemCensus) Remove(when int64, genome Genome) *Cohort {
	var c *Cohort
	func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		k := genome.Hash()
		c = b.Seen[k]
		c.Count -= 1
		b.count -= 1
		if c.Count == 0 {
			delete(b.Seen, k)
			b.distinct -= 1
		}
	}()
	if b.onChange != nil {
		b.onChange(b, c, false)
	}
	return c
}

// Count returns the number of things presently tracked in the world.
func (b *MemCensus) Count() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.count
}

// CountAllTime returns the number of things ever added to the world.
func (b *MemCensus) CountAllTime() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.countAll
}

// Distinct returns the number of distinct genomes currently represented in the world.
func (b *MemCensus) Distinct() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.distinct
}

// DistinctAllTime returns the number of distinct genomes ever seen in the world.
func (b *MemCensus) DistinctAllTime() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.distinctAll
}

// OnChange sets a callback to be invoked for every Add/Remove operation.
func (b *MemCensus) OnChange(fn OnChangeCallback) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.onChange = fn
}
