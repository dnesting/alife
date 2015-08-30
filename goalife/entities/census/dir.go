package census

import "bufio"
import "fmt"
import "io/ioutil"
import "os"
import "path"
import "math/rand"

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
