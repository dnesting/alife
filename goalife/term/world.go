package term

import "io"
import "sort"
import "sync"

import "github.com/dnesting/alife/goalife/world/grid2d"

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

func writeRune(w io.Writer, r rune) {
	io.WriteString(w, string(r))
}

func addHeader(w io.Writer, width int) {
	writeRune(w, topLeftRune)
	for x := 0; x < width; x++ {
		writeRune(w, topRune)
	}
	writeRune(w, topRightRune)
	writeRune(w, '\n')
}

func addFooter(w io.Writer, width int) {
	writeRune(w, bottomLeftRune)
	for x := 0; x < width; x++ {
		writeRune(w, bottomRune)
	}
	writeRune(w, bottomRightRune)
}

func fillBefore(w io.Writer, x, y int, width int, ix, iy *int) {
	for ; *iy < y; *iy++ {
		if *ix == -1 {
			writeRune(w, leftRune)
			*ix = 0
		}
		for ; *ix < width; *ix++ {
			writeRune(w, emptyRune)
		}
		*ix = -1
		writeRune(w, rightRune)
		writeRune(w, '\n')
	}
	if *ix == -1 {
		writeRune(w, leftRune)
		*ix = 0
	}
	for ; *ix < x; *ix++ {
		writeRune(w, emptyRune)
	}
}

type byCoordinate []grid2d.Point

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

var locPool = sync.Pool{New: func() interface{} { return make([]grid2d.Point, 0) }}

func PrintWorld(w io.Writer, g grid2d.Grid) {
	points := locPool.Get().([]grid2d.Point)
	width, height, _ := g.Locations(&points)
	sort.Sort(byCoordinate(points))

	iy, ix := 0, -1
	addHeader(w, width)

	for _, p := range points {
		fillBefore(w, p.X, p.Y, width, &ix, &iy)
		writeRune(w, RuneForOccupant(p.V))
		ix += 1
	}
	locPool.Put(points)
	fillBefore(w, width, height-1, width, &ix, &iy)
	writeRune(w, rightRune)
	writeRune(w, '\n')
	addFooter(w, width)
}
