package term

import "io"
import "sort"
import "time"

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

func PrintWorld(w io.Writer, g grid2d.Grid, fn func(interface{}) rune) {
	width, height, locs := g.Locations()
	printWorld(w, locs, width, height, fn)
}

func printWorld(w io.Writer, points []grid2d.Point, width, height int, fn RuneFunc) {
	if fn == nil {
		fn = DefaultRunes
	}
	sort.Sort(byCoordinate(points))

	iy, ix := 0, -1
	addHeader(w, width)

	for _, p := range points {
		fillBefore(w, p.X, p.Y, width, &ix, &iy)
		writeRune(w, fn(p.V))
		ix += 1
	}
	fillBefore(w, width, height-1, width, &ix, &iy)
	writeRune(w, rightRune)
	writeRune(w, '\n')
	addFooter(w, width)
}

func Printer(w io.Writer, g grid2d.Grid, fn func(interface{}) rune, tty bool, minFreq time.Duration) {
	var due time.Time
	var timeCh <-chan time.Time
	updateCh := make(chan []grid2d.Update, 0)

	g.Subscribe(updateCh)
	defer g.Unsubscribe(updateCh)

	var locs []grid2d.Point
	var width, height int

	doPrint := func(now time.Time) {
		due = now.Add(minFreq)
		if tty {
			io.WriteString(w, "[H")
		}
		printWorld(w, locs, width, height, fn)
		locs = nil
		timeCh = nil
	}

	doUpdate := func() {
		width, height, locs = g.Locations()
		now := time.Now()
		if due.Before(now) {
			doPrint(now)
		} else {
			if timeCh == nil {
				timeCh = time.After(due.Sub(now))
			}
		}
	}

	doUpdate()

	for {
		select {
		case u := <-updateCh:
			if u == nil {
				return
			}
			doUpdate()
		case <-timeCh:
			doPrint(time.Now())
		}
	}
}
