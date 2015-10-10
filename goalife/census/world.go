package census

import "github.com/dnesting/alife/goalife/world/grid2d"

func WatchWorld(c Census, ch <-chan []grid2d.Update, timeFn func() interface{}, keyFn func(interface{}) *Key) {
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
