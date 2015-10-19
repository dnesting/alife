package grid2d

import "github.com/dnesting/alife/goalife/census"

func ScanForCensus(c census.Census, g Grid, timeFn func() interface{}, keyFn func(interface{}) *census.Key) {
	var locs []Point
	g.Locations(&locs)

	for _, p := range locs {
		if key := keyFn(p.V); key != nil {
			c.Add(timeFn(), *key)
		}
	}
}

func WatchForCensus(c census.Census, g Grid, ch <-chan []Update, timeFn func() interface{}, keyFn func(interface{}) *census.Key) {
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
