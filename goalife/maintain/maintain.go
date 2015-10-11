package maintain

import "github.com/dnesting/alife/goalife/world/grid2d"
import "github.com/dnesting/alife/goalife/log"

var Logger = log.Null()

func Maintain(ch <-chan []grid2d.Update, counterFn func(o interface{}) bool, fn func(), keep int) {
	Logger.Printf("seeding initial %d items\n", keep)
	go func() {
		for i := 0; i < keep; i++ {
			fn()
		}
	}()

	var count int
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
