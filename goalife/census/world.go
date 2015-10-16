package census

import "encoding/gob"
import "time"

import "github.com/dnesting/alife/goalife/grid2d"
import "github.com/dnesting/alife/goalife/grid2d/food"
import "github.com/dnesting/alife/goalife/grid2d/org/driver/cpu1"

func RegisterGobTypes() {
	gob.Register(&cpu1.Cpu{})
	gob.Register(&food.Food{})
	gob.Register(time.Time{})
}

func ScanWorld(c Census, g grid2d.Grid, timeFn func() interface{}, keyFn func(interface{}) *Key) {
	var locs []grid2d.Point
	g.Locations(&locs)

	for _, p := range locs {
		if key := keyFn(p.V); key != nil {
			c.Add(timeFn(), *key)
		}
	}
}

func WatchWorld(c Census, g grid2d.Grid, ch <-chan []grid2d.Update, timeFn func() interface{}, keyFn func(interface{}) *Key) {
	RegisterGobTypes()
	ScanWorld(c, g, timeFn, keyFn)

	for updates := range ch {
		if updates == nil {
			return
		}
		for _, u := range updates {
			if u.IsAdd() || u.IsReplace() {
				if key := keyFn(u.New.V); key != nil {
					c.Add(timeFn(), *key)
				}
			}
			if u.IsRemove() || u.IsReplace() {
				if key := keyFn(u.Old.V); key != nil {
					c.Remove(timeFn(), *key)
				}
			}
		}
	}
}
