package grid2d

import "github.com/dnesting/alife/goalife/census"

// ScanForCensus calls c.Add for each occupant of g with the time
// provided by timeFn and census.Key provided by keyFn.  This is
// used to populate a Census from a pre-existing Grid.
func ScanForCensus(c census.Census, g Grid, timeFn func(interface{}) interface{}, keyFn func(interface{}) *census.Key) {
	var locs []Point
	g.Locations(&locs)

	for _, p := range locs {
		if key := keyFn(p.V); key != nil {
			c.Add(timeFn(p.V), *key)
		}
	}
}

// WatchForCensus monitors ch and invokes c.Add and c.Remove as
// appropriate with the time provided by timeFn and census.Key
// provided by keyFn.  If keyFn returns nil, no event will be recorded.
func WatchForCensus(c census.Census, ch <-chan []Update, timeFn func(interface{}) interface{}, keyFn func(interface{}) *census.Key) {
	for updates := range ch {
		for _, u := range updates {
			if u.IsAdd() || u.IsReplace() {
				if key := keyFn(u.New.V); key != nil {
					c.Add(timeFn(u.New.V), *key)
				}
			}
			if u.IsRemove() || u.IsReplace() {
				if key := keyFn(u.Old.V); key != nil {
					c.Remove(timeFn(u.Old.V), *key)
				}
			}
		}
	}
}
