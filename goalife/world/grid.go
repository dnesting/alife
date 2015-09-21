// Package world defines the world within which organisms and other occupants will exist.
package world

import "bytes"
import "encoding/gob"
import "fmt"

type Grid struct {
	height, width int
	data          []interface{}
}

func NewGrid(width, height int) *Grid {
	return &Grid{
		width:  width,
		height: height,
		data:   make([]interface{}, width*height),
	}
}

func (d *Grid) Height() int {
	return d.height
}

func (d *Grid) Width() int {
	return d.width
}

func modPos(v, max int) int {
	v %= max
	if v < 0 {
		v += max
	}
	return v
}

// Wrap ensures the x and y coordinates are in-bounds
func (d *Grid) Wrap(x, y int) (int, int) {
	x = modPos(x, d.width)
	y = modPos(y, d.height)

	return x, y
}

func (d *Grid) checkRange(x, y int) {
	if x < 0 || x >= d.width || y < 0 || y >= d.height {
		panic(fmt.Sprintf("coordinate (%d,%d) outside of %v", x, y, d))
	}
}

// offset converts the (x, y) coordinates to a slice offset.  The given
// coordinates can be outside of the (width, height) ranges for the world,
// which will just result in the location wrapping around.
func (d *Grid) offset(x, y int) int {
	d.checkRange(x, y)
	return modPos(y*d.width+x, d.height*d.width)
}

func (d *Grid) Copy() *Grid {
	n := NewGrid(d.width, d.height)
	copy(n.data, d.data)
	return n
}

func (d *Grid) Get(x, y int) interface{} {
	return d.data[d.offset(x, y)]
}

func (d *Grid) Put(x, y int, o interface{}) (orig interface{}) {
	i := d.offset(x, y)
	orig = d.data[i]
	d.data[i] = o
	return
}

func (d *Grid) Each(fn func(x, y int, value interface{})) {
	for y := 0; y < d.height; y++ {
		for x := 0; x < d.width; x++ {
			i := y*d.width + x
			o := d.data[i]
			if o != nil {
				fn(x, y, o)
			}
		}
	}
}

func (d *Grid) Resized(width, height int, dropFn func(x, y int, v interface{})) *Grid {
	n := NewGrid(width, height)
	d.Each(func(x, y int, value interface{}) {
		if x < width && y < height {
			n.Put(x, y, value)
		} else if dropFn != nil {
			dropFn(x, y, value)
		}
	})
	return n
}

func (d *Grid) String() string {
	return fmt.Sprintf("[data %dx%d]", d.width, d.height)
}

type gobItem struct {
	X, Y int
	V    interface{}
}

func (d *Grid) GobEncode() ([]byte, error) {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	if err := enc.Encode(d.height); err != nil {
		return nil, err
	}
	if err := enc.Encode(d.width); err != nil {
		return nil, err
	}
	var items []gobItem
	d.Each(func(x, y int, v interface{}) {
		items = append(items, gobItem{x, y, v})
	})
	if err := enc.Encode(items); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (d *Grid) GobDecode(stream []byte) error {
	dec := gob.NewDecoder(bytes.NewReader(stream))
	if err := dec.Decode(&d.height); err != nil {
		return err
	}
	if err := dec.Decode(&d.width); err != nil {
		return err
	}
	d.data = make([]interface{}, d.width*d.height)
	var items []gobItem
	if err := dec.Decode(&items); err != nil {
		return err
	}
	for _, i := range items {
		d.Put(i.X, i.Y, i.V)
	}
	return nil
}
