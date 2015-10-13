package census

import "fmt"
import "sync"

// MemCensus implements a Census entirely in-memory.
type MemCensus struct {
	mu          sync.RWMutex
	seen        map[uint64]*Population
	count       int
	countAll    int
	distinct    int
	distinctAll int
}

func (b *MemCensus) Get(key Key) (Population, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	c, ok := b.seen[key.Hash()]
	if ok {
		return *c, true
	}
	return Population{}, false
}

// Add indicates an instance of the given key was added to the world.
func (b *MemCensus) Add(when interface{}, key Key) (ret Population) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.seen == nil {
		b.seen = make(map[uint64]*Population)
	}

	h := key.Hash()
	c, ok := b.seen[h]
	if !ok {
		c = &Population{
			Key:   key,
			First: when,
		}
		b.seen[h] = c
		b.distinct += 1
		b.distinctAll += 1
	}
	c.Count += 1
	b.count += 1
	b.countAll += 1
	return *c
}

// Remove indicates an instance of the given key was removed from the world.
func (b *MemCensus) Remove(when interface{}, key Key) (ret Population) {
	b.mu.Lock()
	defer b.mu.Unlock()

	h := key.Hash()
	c, ok := b.seen[h]
	if ok {
		c.Count -= 1
		b.count -= 1
		if c.Count == 0 {
			delete(b.seen, h)
			b.distinct -= 1
			c.Last = when
		}
		return *c
	}
	panic(fmt.Sprintf("mismatched remove for %v", key))
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

// Distinct returns the number of distinct keys currently represented in the world.
func (b *MemCensus) Distinct() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.distinct
}

// DistinctAllTime returns the number of distinct keys ever seen in the world.
func (b *MemCensus) DistinctAllTime() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.distinctAll
}
