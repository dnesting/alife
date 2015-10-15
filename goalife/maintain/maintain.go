package maintain

import "github.com/dnesting/alife/goalife/world/grid2d"
import "github.com/dnesting/alife/goalife/log"

var Logger = log.Null()

func Count(g grid2d.Grid, counterFn func(o interface{}) bool) int {
	var locs []grid2d.Point
	g.Locations(&locs)

	var count int
	for _, p := range locs {
		if counterFn(p.V) {
			count += 1
		}
	}
	return count
}

func Maintain(g grid2d.Grid, ch <-chan []grid2d.Update, counterFn func(o interface{}) bool, fn func(), keep int, initial int) {
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
