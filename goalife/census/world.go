package census

import "github.com/dnesting/alife/goalife/world/grid2d"

func WatchWorld(c Census, g grid2d.Grid, timeFn func() interface{}, keyFn func(interface{}) *Key, ready chan<- bool) {
	updateCh := make(chan []grid2d.Update)
	g.Subscribe(updateCh)
	if ready != nil {
		ready <- true
	}
	defer g.Unsubscribe(updateCh)

	for updates := range updateCh {
		if updates == nil {
			return
		}
		for _, u := range updates {
			if u.IsAdd() || u.IsReplace() {
				if key := keyFn(u.New); key != nil {
					c.Add(timeFn(), *key)
				}
			}
			if u.IsRemove() || u.IsReplace() {
				if key := keyFn(u.Old); key != nil {
					c.Remove(timeFn(), *key)
				}
			}
		}
	}
}
