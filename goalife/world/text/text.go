// Package world defines the world within which organisms and other occupants will exist.
package text

import "bytes"
import "sort"

import "github.com/dnesting/alife/goalife/world"

const (
	topLeftRune     = '┌'
	topRune         = '─'
	topRightRune    = '┐'
	rightRune       = '│'
	bottomRightRune = '┘'
	bottomRune      = '─'
	bottomLeftRune  = '└'
	leftRune        = '│'
	emptyRune       = ' '
)

func addHeader(b *bytes.Buffer, width int) {
	b.WriteRune(topLeftRune)
	for x := 0; x < width; x++ {
		b.WriteRune(topRune)
	}
	b.WriteRune(topRightRune)
	b.WriteRune('\n')
}

func addFooter(b *bytes.Buffer, width int) {
	b.WriteRune(bottomLeftRune)
	for x := 0; x < width; x++ {
		b.WriteRune(bottomRune)
	}
	b.WriteRune(bottomRightRune)
}

func fillBefore(b *bytes.Buffer, x, y int, width int, ix, iy *int) {
	for ; *iy < y; *iy++ {
		if *ix == -1 {
			b.WriteRune(leftRune)
			*ix = 0
		}
		for ; *ix < width; *ix++ {
			b.WriteRune(emptyRune)
		}
		*ix = -1
		b.WriteRune(rightRune)
		b.WriteRune('\n')
	}
	if *ix == -1 {
		b.WriteRune(leftRune)
		*ix = 0
	}
	for ; *ix < x; *ix++ {
		b.WriteRune(emptyRune)
	}
}

type point struct {
	X, Y int
}

type byCoordinate []point

func (p byCoordinate) Len() int { return len(p) }
func (p byCoordinate) Less(i, j int) bool {
	if p[i].Y < p[j].Y {
		return true
	}
	if p[i].Y == p[j].Y && p[i].X < p[j].X {
		return true
	}
	return false
}
func (p byCoordinate) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

func WorldAsString(w *world.World) string {
	b := bytes.Buffer{}
	var points []point
	m := make(map[point]rune)

	w.Each(func(x, y int, o world.Occupant) {
		p := point{x, y}
		m[p] = OccupantAsRune(o)
		points = append(points, p)
	})

	sort.Sort(byCoordinate(points))

	iy, ix := 0, -1
	addHeader(&b, w.Width)

	for _, p := range points {
		fillBefore(&b, p.X, p.Y, w.Width, &ix, &iy)
		b.WriteRune(m[p])
		ix += 1
	}
	fillBefore(&b, w.Width, w.Height-1, w.Width, &ix, &iy)
	b.WriteRune(rightRune)
	b.WriteRune('\n')
	addFooter(&b, w.Width)
	return b.String()
}

type Printable interface {
	Rune() rune
}

func OccupantAsRune(o interface{}) rune {
	switch o := o.(type) {
	case Printable:
		return o.Rune()
	default:
		return '?'
	}
}
