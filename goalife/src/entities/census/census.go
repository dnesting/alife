// Package census implements a method for tracking genomes.
//
// A Genome consists of (a) an identifying uint32 hash and (b) a []string
// describing the full contents of the genome.  The meaning and derivation
// of these two items is implementation-dependent.
package census

import "bufio"
import "fmt"
import "io/ioutil"
import "os"
import "path"
import "math/rand"
import "sync"

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

// DirCensus implements a Census that saves interesting genomes to disk.
// This type wraps a MemCensus and behaves similarly.
type DirCensus struct {
	MemCensus
	Dir         string // the parent directory holding genomes
	NumRecorded int    // the number of genomes written to disk
	threshold   int    // the population threshold for writing a genome to disk
}

// NewDirCensus creates a new DirCensus writing to the given dir any genome
// that appears more than threshold times in the world.
func NewDirCensus(dir string, threshold int) *DirCensus {
	return &DirCensus{
		MemCensus: MemCensus{
			Seen: make(map[uint32]*Cohort),
		},
		Dir:       dir,
		threshold: threshold,
	}
}

func (b *DirCensus) filename(c *Cohort) string {
	return path.Join(b.Dir, fmt.Sprintf("%d.%d", c.First, c.Genome.Hash()))
}

// PreviouslyRecorded returns true if the given Cohort was previously written to disk.
func (b *DirCensus) PreviouslyRecorded(c *Cohort) bool {
	_, err := os.Stat(b.filename(c))
	return err == nil
}

// RecordInDir writes the given cohort to disk.
func (b *DirCensus) RecordInDir(c *Cohort) error {
	f, err := os.Create(b.filename(c))
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	w.WriteString(fmt.Sprintf("First: %d\n", c.First))
	w.WriteString(fmt.Sprintf("Last: %d\n", c.Last))
	code := c.Genome.Code()
	if len(code) > 0 {
		w.WriteString("Code:\n")
		for _, s := range code {
			w.WriteString(s)
			w.WriteString("\n")
		}
	}
	if err := w.Flush(); err != nil {
		return err
	}
	return nil
}

// fileGenome contains basic genome data as retrieved from disk.
type fileGenome struct {
	hash uint32
	code []string
}

func (g *fileGenome) Hash() uint32 {
	return g.hash
}

func (g *fileGenome) Code() []string {
	return g.code
}

// Random retrieves a randomly-selected Cohort from disk.
func (b *DirCensus) Random() (*Cohort, error) {
	ls, err := ioutil.ReadDir(b.Dir)
	if err != nil {
		return nil, err
	}
	if len(ls) == 0 {
		return nil, nil
	}
	fi := ls[rand.Intn(len(ls))]
	name := path.Join(b.Dir, fi.Name())
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scan := bufio.NewScanner(f)

	var coding bool
	var code []string
	for scan.Scan() {
		if coding {
			code = append(code, scan.Text())
		} else if scan.Text() == "Code:" {
			coding = true
		}
	}

	return &Cohort{
		Genome: &fileGenome{
			code: code,
		},
	}, nil
}

// Add indicates an instance of the given genome was added to the world,
// possibly writing the Cohort to disk if it exceeds the DirCensus's threshold.
func (b *DirCensus) Add(when int64, genome Genome) *Cohort {
	c := b.MemCensus.Add(when, genome)

	if c.Count >= b.threshold && !b.PreviouslyRecorded(c) && len(c.Genome.Code()) > 0 {
		b.RecordInDir(c)
		b.NumRecorded++
	}
	return c
}

// Add indicates an instance of the given genome was removed from the world,
// possibly writing the Cohort to disk to record its last-seen information if
// it was previously written there.
func (b *DirCensus) Remove(when int64, genome Genome) *Cohort {
	c := b.MemCensus.Remove(when, genome)

	if c.Count == 0 && b.PreviouslyRecorded(c) {
		// Capture last frame info for extinct species
		c.Last = when
		b.RecordInDir(c)
	}
	return c
}
