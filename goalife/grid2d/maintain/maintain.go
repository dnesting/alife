// Package maintain keeps a minimum number of items in a grid.
package maintain

import "github.com/dnesting/alife/goalife/grid2d"
import "github.com/dnesting/alife/goalife/log"

var Logger = log.Null()

// CounterFunc identifies whether an item should contribute to the count of items
// in the grid, for the purposes of deciding whether to add more.
type CounterFunc func(o interface{}) bool

// Count counts the number of occupants in g that satisfy fn.
func Count(g grid2d.Grid, fn CounterFunc) int {
	var locs []grid2d.Point
	g.Locations(&locs)

	var count int
	for _, p := range locs {
		if fn(p.V) {
			count += 1
		}
	}
	return count
}

// Maintain watches ch and invokes fn to keep the number of counted occupants above keep.
// Only occupants satisfying counterFn will be counted.  Invoking fn must (eventually) result
// in increasing the count of occupants satisfying counterFn by at least 1.
func Maintain(ch <-chan []grid2d.Update, counterFn func(o interface{}) bool, fn func(), keep int, initial int) {
	count := initial

	if count < keep {
		Logger.Printf("seeding %d items to get up to %d\n", keep-count, keep)

		go func(count int) {
			for i := count; i < keep; i++ {
				fn()
			}
		}(count)
	}

	for updates := range ch {
		for _, u := range updates {
			if u.IsAdd() || u.IsReplace() {
				if counterFn(u.New.V) {
					count++
					Logger.Printf("%v added, count %d\n", u.New.V, count)
				}
			}
			if u.IsRemove() || u.IsReplace() {
				if counterFn(u.Old.V) {
					count--
					if count < keep {
						Logger.Printf("%v removed, count %d, adding one\n", u.Old.V, count)
						go fn()
					} else {
						Logger.Printf("%v removed, count %d\n", u.Old.V, count)
					}
				}
			}
		}
	}
}
