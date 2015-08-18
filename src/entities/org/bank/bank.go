package bank

import "bufio"
import "fmt"
import "hash/crc32"
import "os"
import "path"

type Cohort struct {
	Genome uint32
	Code   []byte
	Count  int
	First  int
	Last   int
}

type Survey struct {
	Seen  map[uint32]*Cohort
	count int
}

func NewSurvey() *Survey {
	return &Survey{
		Seen: make(map[uint32]*Cohort),
	}
}

func (s *Survey) Record(code []byte) {
	h := crc32.ChecksumIEEE(code)
	if _, ok := s.Seen[h]; !ok {
		s.Seen[h] = new(Cohort)
		s.Seen[h].Genome = h
		s.Seen[h].Code = code
	}
	s.Seen[h].Count += 1
	s.count += 1
}

func (s *Survey) Count() int {
	return s.count
}

func (s *Survey) Distinct() int {
	return len(s.Seen)
}

type Bank interface {
	Last() *Survey
	Record(frame int, s *Survey)
}

type MemBank struct {
	last *Survey
}

func (b *MemBank) Last() *Survey {
	return b.last
}

func (b *MemBank) Record(frame int, s *Survey) {
	for k, v := range s.Seen {
		v.First = frame
		if b.last != nil {
			if p, ok := b.last.Seen[k]; ok {
				v.First = p.First
			}
		}
		v.Last = -1
	}
	b.last = s
}

type DirBank struct {
	MemBank
	Dir         string
	NumRecorded int
}

func NewDirBank(dir string) *DirBank {
	return &DirBank{Dir: dir}
}

const DirRecordThreshold = 10

func (b *DirBank) filename(c Cohort) string {
	return path.Join(b.Dir, fmt.Sprintf("%d.%d", c.First, c.Genome))
}

func (b *DirBank) PreviouslyRecorded(c Cohort) bool {
	_, err := os.Stat(b.filename(c))
	return err == nil
}

func (b *DirBank) RecordInDir(c Cohort) error {
	fmt.Printf("archiving %s\n", b.filename(c))
	f, err := os.Create(b.filename(c))
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	w.WriteString(fmt.Sprintf("First: %d\n", c.First))
	w.WriteString(fmt.Sprintf("Last: %d\n", c.Last))
	w.WriteString(fmt.Sprintf("Code: %v\n", c.Code))
	if err := w.Flush(); err != nil {
		return err
	}
	return nil
}

func (b *DirBank) Record(frame int, s *Survey) {
	l := b.MemBank.Last()
	b.MemBank.Record(frame, s)

	for _, v := range s.Seen {
		if v.Count >= DirRecordThreshold && !b.PreviouslyRecorded(*v) {
			b.RecordInDir(*v)
			b.NumRecorded++
		}
	}

	if l != nil {
		// Capture last frame info for extinct species
		for k, v := range l.Seen {
			if _, ok := s.Seen[k]; !ok {
				if b.PreviouslyRecorded(*v) {
					v.Last = frame - 1
					b.RecordInDir(*v)
				}
			}
		}
	}
}
