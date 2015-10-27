package census

import "errors"
import "encoding/gob"
import "fmt"
import "io"
import "io/ioutil"
import "math/rand"
import "os"
import "path"

var deps = struct {
	ReadDir  func(string) ([]os.FileInfo, error)
	Stat     func(string) (os.FileInfo, error)
	Create   func(string) (io.ReadWriteCloser, error)
	Open     func(string) (io.ReadWriteCloser, error)
	MkdirAll func(string, os.FileMode) error
}{
	ioutil.ReadDir,
	os.Stat,
	func(s string) (io.ReadWriteCloser, error) { return os.Create(s) },
	func(s string) (io.ReadWriteCloser, error) { return os.Open(s) },
	os.MkdirAll,
}

// DirCensus implements a Census that saves interesting populations to disk.
type DirCensus struct {
	Dir       string                  // the parent directory holding populations
	Threshold func(p Population) bool // the deciding func for whether an Add should be persistent
	MemCensus
	numRecorded int // the number of populations written to disk
}

// NewDirCensus creates a DirCensus storing populations that satisfy
// threshold in dir.
func NewDirCensus(dir string, threshold func(p Population) bool) (*DirCensus, error) {
	b := &DirCensus{
		Dir:       dir,
		Threshold: threshold,
	}
	if err := deps.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	ls, _ := deps.ReadDir(b.Dir)
	b.numRecorded = len(ls)
	return b, nil
}

func (b *DirCensus) filename(key Key) string {
	return path.Join(b.Dir, fmt.Sprintf("%x", key.Hash()))
}

// GetFromRecord retrieves the population with key from disk.
func (b *DirCensus) GetFromRecord(key Key) (Population, error) {
	return b.decodeFromFilename(b.filename(key))
}

// IsRecorded returns true if a population with key exists on disk.
func (b *DirCensus) IsRecorded(key Key) bool {
	_, err := deps.Stat(b.filename(key))
	return err == nil
}

// Record writes population to disk.
func (b *DirCensus) Record(c Population) error {
	f, err := deps.Create(b.filename(c.Key))
	if err != nil {
		return err
	}
	defer f.Close()

	enc := gob.NewEncoder(f)
	if err := enc.Encode(c); err != nil {
		return err
	}
	return nil
}

var ErrNoneFound = errors.New("none found")

// Random retrieves a randomly-selected Population from disk.
func (b *DirCensus) Random() (Population, error) {
	ls, err := deps.ReadDir(b.Dir)
	if err != nil {
		return Population{}, err
	}
	if len(ls) == 0 {
		return Population{}, ErrNoneFound
	}
	fi := ls[rand.Intn(len(ls))]
	return b.decodeFromFilename(path.Join(b.Dir, fi.Name()))
}

func (b *DirCensus) decodeFromFilename(name string) (Population, error) {
	f, err := deps.Open(name)
	if err != nil {
		return Population{}, err
	}
	defer f.Close()

	dec := gob.NewDecoder(f)
	var p Population
	if err := dec.Decode(&p); err != nil {
		return Population{}, err
	}
	return p, nil
}

// Add indicates an instance of population was added, possibly
// writing the Population to disk if it satisfies the DirCensus's
// threshold.
func (b *DirCensus) Add(when interface{}, key Key) Population {
	c := b.MemCensus.Add(when, key)

	if (b.Threshold == nil || b.Threshold(c)) && !b.IsRecorded(key) {
		b.Record(c)
		b.numRecorded++
	}
	return c
}

// Remove indicates an instance of population was removed, possibly
// writing the Population to disk to record its last-seen information
// if it was previously written there.
func (b *DirCensus) Remove(when interface{}, key Key) Population {
	c := b.MemCensus.Remove(when, key)

	if c.Count == 0 && b.IsRecorded(c.Key) {
		b.Record(c)
	}
	return c
}

// NumRecorded returns the number of populations currently seen in dir.
func (b *DirCensus) NumRecorded() int {
	return b.numRecorded
}
