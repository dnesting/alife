package world

import "bytes"
import "sync"

type Occupant interface{}

type World interface {
	At(x, y int) Occupant
	Put(x, y int, o Occupant) Occupant
	PutIfEmpty(x, y int, o Occupant) Occupant
	Copy() World
	Each(fn func(x, y int, o Occupant))
}

type world struct {
	Height, Width int

	mu   sync.RWMutex
	data []Occupant
}

func (w *world) offset(x, y int) int {
	return y*w.Width + x
}

func (w *world) At(x, y int) Occupant {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.data[w.offset(x, y)]
}

func (w *world) Put(x, y int, o Occupant) Occupant {
	w.mu.Lock()
	defer w.mu.Unlock()

	offset := w.offset(x, y)
	old := w.data[offset]
	w.data[offset] = o
	return old
}

func (w *world) PutIfEmpty(x, y int, o Occupant) Occupant {
	w.mu.Lock()
	defer w.mu.Unlock()

	offset := w.offset(x, y)
	if w.data[offset] == nil {
		w.data[offset] = o
		return nil
	}
	return w.data[offset]
}

func (w *world) Copy() World {
	w.mu.RLock()
	defer w.mu.RUnlock()

	data := make([]Occupant, w.Height*w.Width)
	copy(data, w.data)

	return &world{
		Height: w.Height,
		Width:  w.Width,
		data:   data,
	}
}

func (w *world) Each(fn func(x, y int, o Occupant)) {
	for y := 0; y < w.Height; y++ {
		for x := 0; x < w.Width; x++ {
			if i := w.At(x, y); i != nil {
				fn(x, y, i)
			}
		}
	}
}

type Printable interface {
	Rune() rune
}

func (w *world) String() string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	b := bytes.Buffer{}
	headFoot := func() {
		b.WriteString("+")
		for x := 0; x < w.Width; x++ {
			b.WriteString("-")
		}
		b.WriteString("+\n")
	}

	headFoot()
	for y := 0; y < w.Height; y++ {
		b.WriteString("|")
		for x := 0; x < w.Width; x++ {
			i := w.At(x, y)
			if i == nil {
				b.WriteString(" ")
			} else {
				switch i := i.(type) {
				case Printable:
					b.WriteRune(i.Rune())
				default:
					b.WriteString("?")
				}
			}
		}
		b.WriteString("|\n")
	}
	headFoot()
	return b.String()
}

func New(h, w int) World {
	return &world{
		Height: h,
		Width:  w,
		data:   make([]Occupant, h*w),
	}
}
