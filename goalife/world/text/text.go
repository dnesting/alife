// Package world defines the world within which organisms and other occupants will exist.
package text

import "bytes"

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

func WorldAsString(w *world.World) string {
	b := bytes.Buffer{}

	iy, ix := 0, -1
	addHeader(&b, w.Width)

	w.Each(func(x, y int, o world.Occupant) {
		fillBefore(&b, x, y, w.Width, &ix, &iy)
		b.WriteRune(OccupantAsRune(o))
		ix += 1
	})
	fillBefore(&b, w.Width, w.Height-1, w.Width, &ix, &iy)
	b.WriteRune(rightRune)
	b.WriteRune('\n')
	addFooter(&b, w.Width)
	return b.String()
}

type Printable interface {
	Rune() rune
}

func OccupantAsRune(o world.Occupant) rune {
	switch o := o.(type) {
	case Printable:
		return o.Rune()
	default:
		return '?'
	}
}
