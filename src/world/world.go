package world

import "bytes"
import "math/rand"
import "sync"

type Occupant interface{}

type World interface {
	ConsiderEmpty(func(o Occupant) bool)
	At(x, y int) Occupant
	Put(x, y int, o Occupant) Occupant
	PutIfEmpty(x, y int, o Occupant) Occupant
	PutRandomlyIfEmpty(o Occupant) Occupant
	RemoveIfEqual(x, y int, o Occupant) Occupant
	ReplaceIfEqual(x, y int, o, n Occupant) Occupant
	MoveIfEmpty(x1, y1, x2, y2 int) Occupant
	Copy() World
	Each(fn func(x, y int, o Occupant))
	Dimensions() (int, int)
}

type world struct {
	Height, Width int

	mu      sync.RWMutex
	data    []Occupant
	emptyFn func(o Occupant) bool
}

func (w *world) ConsiderEmpty(fn func(o Occupant) bool) {
	w.emptyFn = fn
}

func (w *world) isEmpty(o Occupant) bool {
	if o == nil {
		return true
	}
	if w.emptyFn == nil {
		return false
	}
	return w.emptyFn(o)
}

func (w *world) Dimensions() (int, int) {
	return w.Width, w.Height
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

func (w *world) PutRandomlyIfEmpty(o Occupant) Occupant {
	width, height := w.Dimensions()
	return w.PutIfEmpty(rand.Intn(width), rand.Intn(height), o)
}

func (w *world) PutIfEmpty(x, y int, o Occupant) Occupant {
	w.mu.Lock()
	defer w.mu.Unlock()

	offset := w.offset(x, y)
	if w.isEmpty(w.data[offset]) {
		w.data[offset] = o
		return nil
	}
	return w.data[offset]
}

func (w *world) MoveIfEmpty(x1, y1, x2, y2 int) Occupant {
	w.mu.Lock()
	defer w.mu.Unlock()

	o1 := w.offset(x1, y1)
	o2 := w.offset(x2, y2)

	// If source cell is empty, don't move anything (return success)
	if w.data[o1] == nil {
		return nil
	}
	// If dest cell isn't empty, don't move anything (return its occupant)
	if w.data[o2] != nil {
		return w.data[o2]
	}
	w.data[o2] = w.data[o1]
	w.data[o1] = nil
	return nil
}

func (w *world) RemoveIfEqual(x, y int, o Occupant) Occupant {
	return w.ReplaceIfEqual(x, y, o, nil)
}

func (w *world) ReplaceIfEqual(x, y int, o Occupant, n Occupant) Occupant {
	w.mu.Lock()
	defer w.mu.Unlock()

	offset := w.offset(x, y)
	orig := w.data[offset]
	if orig == o {
		w.data[offset] = n
	}
	return orig
}

func (w *world) Copy() World {
	w.mu.RLock()
	defer w.mu.RUnlock()

	data := make([]Occupant, w.Height*w.Width)
	copy(data, w.data)

	return &world{
		Height:  w.Height,
		Width:   w.Width,
		data:    data,
		emptyFn: w.emptyFn,
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
