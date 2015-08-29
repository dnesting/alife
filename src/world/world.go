package world

import "bytes"
import "encoding/gob"
import "math/rand"
import "sync"

type Occupant interface{}

type World interface {
	ConsiderEmpty(func(o Occupant) bool)
	At(x, y int) Occupant
	Put(x, y int, o Occupant) Occupant
	PutIfEmpty(x, y int, o Occupant) Occupant
	PlaceRandomly(o Occupant) (int, int)
	Remove(x, y int) Occupant
	RemoveIfEqual(x, y int, o Occupant) Occupant
	ReplaceIfEqual(x, y int, o, n Occupant) Occupant
	MoveIfEmpty(x1, y1, x2, y2 int) Occupant
	Copy() World
	Each(fn func(x, y int, o Occupant))
	Dimensions() (int, int)
	OnUpdate(fn func(w World))
}

type BasicWorld struct {
	Height, Width int

	mu       sync.RWMutex
	data     []Occupant
	emptyFn  func(o Occupant) bool
	updateFn func(w World)
}

func (w *BasicWorld) GobEncode() ([]byte, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	if err := enc.Encode(w.Height); err != nil {
		return nil, err
	}
	if err := enc.Encode(w.Width); err != nil {
		return nil, err
	}
	if err := enc.Encode(w.data); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (w *BasicWorld) GobDecode(stream []byte) error {
	dec := gob.NewDecoder(bytes.NewReader(stream))
	if err := dec.Decode(&w.Height); err != nil {
		return err
	}
	if err := dec.Decode(&w.Width); err != nil {
		return err
	}
	if err := dec.Decode(&w.data); err != nil {
		return err
	}
	return nil
}

func (w *BasicWorld) ConsiderEmpty(fn func(o Occupant) bool) {
	w.emptyFn = fn
}

func (w *BasicWorld) OnUpdate(fn func(w World)) {
	w.updateFn = fn
}

func (w *BasicWorld) isEmpty(o Occupant) bool {
	if o == nil {
		return true
	}
	if w.emptyFn == nil {
		return false
	}
	return w.emptyFn(o)
}

func clip(v, max int) int {
	v %= max
	if v < 0 {
		v += max
	}
	return v
}

func (w *BasicWorld) Dimensions() (int, int) {
	return w.Width, w.Height
}

func (w *BasicWorld) offset(x, y int) int {
	return clip(y*w.Width+x, w.Height*w.Width)
}

func (w *BasicWorld) notifyUpdate() {
	if w.updateFn != nil {
		w.updateFn(w)
	}
}

func (w *BasicWorld) At(x, y int) Occupant {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.data[w.offset(x, y)]
}

func (w *BasicWorld) Put(x, y int, o Occupant) Occupant {
	defer w.notifyUpdate()
	w.mu.Lock()
	defer w.mu.Unlock()

	offset := w.offset(x, y)
	old := w.data[offset]
	w.data[offset] = o
	return old
}

func (w *BasicWorld) PlaceRandomly(o Occupant) (int, int) {
	width, height := w.Dimensions()
	var x, y int
	for {
		x, y = rand.Intn(width), rand.Intn(height)
		if w.PutIfEmpty(x, y, o) == nil {
			break
		}
	}
	return x, y
}

func (w *BasicWorld) PutIfEmpty(x, y int, o Occupant) Occupant {
	defer w.notifyUpdate()
	w.mu.Lock()
	defer w.mu.Unlock()

	offset := w.offset(x, y)
	if w.isEmpty(w.data[offset]) {
		w.data[offset] = o
		return nil
	}
	return w.data[offset]
}

func (w *BasicWorld) MoveIfEmpty(x1, y1, x2, y2 int) Occupant {
	defer w.notifyUpdate()
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

func (w *BasicWorld) RemoveIfEqual(x, y int, o Occupant) Occupant {
	return w.ReplaceIfEqual(x, y, o, nil)
}

func (w *BasicWorld) ReplaceIfEqual(x, y int, o Occupant, n Occupant) Occupant {
	defer w.notifyUpdate()
	w.mu.Lock()
	defer w.mu.Unlock()

	offset := w.offset(x, y)
	orig := w.data[offset]
	if orig == o {
		w.data[offset] = n
	}
	return orig
}

func (w *BasicWorld) Remove(x, y int) Occupant {
	return w.Put(x, y, nil)
}

func (w *BasicWorld) Copy() World {
	w.mu.RLock()
	defer w.mu.RUnlock()

	data := make([]Occupant, w.Height*w.Width)
	copy(data, w.data)

	return &BasicWorld{
		Height:  w.Height,
		Width:   w.Width,
		data:    data,
		emptyFn: w.emptyFn,
	}
}

func (w *BasicWorld) Each(fn func(x, y int, o Occupant)) {
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

func (w *BasicWorld) String() string {
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
			i := w.data[w.offset(x, y)]
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
	return &BasicWorld{
		Height: h,
		Width:  w,
		data:   make([]Occupant, h*w),
	}
}
