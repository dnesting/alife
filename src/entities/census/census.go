package census

import "bufio"
import "fmt"
import "io/ioutil"
import "os"
import "path"
import "math/rand"
import "sync"

type Genomer interface {
	Genome() Genome
}

type Genome interface {
	Hash() uint32
	Code() []string
}

type Cohort struct {
	Genome Genome
	Count  int
	First  int64
	Last   int64
}

func (c *Cohort) String() string {
	return fmt.Sprintf("[cohort %d count=%d (%d-%d)]", c.Genome.Hash(), c.Count, c.First, c.Last)
}

type OnChangeCallback func(b Census, c *Cohort, added bool)

type Census interface {
	Add(when int64, genome Genome) *Cohort
	Remove(when int64, genome Genome) *Cohort
	Count() int
	Distinct() int
	CountAllTime() int
	DistinctAllTime() int
	OnChange(fn OnChangeCallback)
}

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

func NewMemCensus() *MemCensus {
	return &MemCensus{
		Seen: make(map[uint32]*Cohort),
	}
}

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

func (b *MemCensus) Count() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.count
}

func (b *MemCensus) CountAllTime() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.countAll
}

func (b *MemCensus) Distinct() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.distinct
}

func (b *MemCensus) DistinctAllTime() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.distinctAll
}

func (b *MemCensus) OnChange(fn OnChangeCallback) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.onChange = fn
}

type DirCensus struct {
	MemCensus
	Dir         string
	NumRecorded int
	threshold   int
}

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

func (b *DirCensus) PreviouslyRecorded(c *Cohort) bool {
	_, err := os.Stat(b.filename(c))
	return err == nil
}

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

func (b *DirCensus) Add(when int64, genome Genome) *Cohort {
	c := b.MemCensus.Add(when, genome)

	if c.Count >= b.threshold && !b.PreviouslyRecorded(c) && len(c.Genome.Code()) > 0 {
		b.RecordInDir(c)
		b.NumRecorded++
	}
	return c
}

func (b *DirCensus) Remove(when int64, genome Genome) *Cohort {
	c := b.MemCensus.Remove(when, genome)

	if c.Count == 0 && b.PreviouslyRecorded(c) {
		// Capture last frame info for extinct species
		c.Last = when
		b.RecordInDir(c)
	}
	return c
}
