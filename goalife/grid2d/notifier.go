package grid2d

import "sync"

type Update struct {
	Old *Point
	New *Point
}

func (u Update) IsAdd() bool {
	return u.Old == nil && u.New != nil
}

func (u Update) IsRemove() bool {
	return u.Old != nil && u.New == nil
}

func (u Update) IsMove() bool {
	return u.Old != nil && u.New != nil && (u.Old.X != u.New.X || u.Old.Y != u.New.Y)
}

func (u Update) IsReplace() bool {
	return u.Old != nil && u.New != nil && u.Old.V != u.New.V
}

type notifier struct {
	mu   sync.Mutex
	subs []chan<- []Update
}

func newNotifier(done <-chan bool) *notifier {
	n := &notifier{
		subs: make([]chan<- []Update, 0),
	}
	go func() {
		<-done
		n.Done()
	}()
	return n
}

func (n *notifier) Done() {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, ch := range n.subs {
		close(ch)
	}
}

func (n *notifier) Subscribe(ch chan<- []Update) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.subs = append(n.subs, ch)
}

func (n *notifier) Unsubscribe(ch chan<- []Update) {
	n.mu.Lock()
	defer n.mu.Unlock()
	subs := make([]chan<- []Update, 0, len(n.subs))
	for _, s := range n.subs {
		if s != ch {
			subs = append(subs, s)
		}
	}
	n.subs = subs
}

func (n *notifier) RecordAdd(x, y int, value interface{}) {
	n.add([]Update{
		Update{
			New: &Point{x, y, value},
		}})
}

func (n *notifier) RecordRemove(x, y int, value interface{}) {
	n.add([]Update{
		Update{
			Old: &Point{x, y, value},
		}})
}

func (n *notifier) RecordMove(x1, y1, x2, y2 int, value interface{}) {
	n.add([]Update{
		Update{
			Old: &Point{x1, y1, value},
			New: &Point{x2, y2, value},
		}})
}

func (n *notifier) RecordReplace(x, y int, orig, repl interface{}) {
	n.add([]Update{Update{
		Old: &Point{x, y, orig},
		New: &Point{x, y, repl},
	}})
}

func (n *notifier) add(u []Update) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, ch := range n.subs {
		ch <- u
	}
}

// NotifyToInterface effectively provides a type conversion from a
// grid2d notification channel (of type []Update) to interface{} for
// use with queueing and rate-limiting functions in util/chanbuf.
// This introduces some modest overhead since Go doesn't support this
// type of type conversion directly.
func NotifyToInterface(ch <-chan []Update) <-chan interface{} {
	chained := make(chan interface{})
	go func() {
		for x := range ch {
			chained <- x
		}
		close(chained)
	}()
	return chained
}

// NotifyFromInterface effectively provides a type conversion from the
// aggregated []interface{} type used by the queueing and rate-limiting
// functions in util/chanbuf, to  the grid2d notification channel (of
// type []Update).  In the process, de-aggregates the aggregated
// messages, potentially sending multiple messages to the returned
// channel for every one message received by ch. This introduces some
// modest overhead since Go doesn't support this type of type conversion
// directly.
func NotifyFromInterface(ch <-chan []interface{}) <-chan []Update {
	chained := make(chan []Update)
	go func() {
		for x := range ch {
			if x != nil {
				for _, y := range x {
					chained <- y.([]Update)
				}
			} else {
				chained <- nil
			}
		}
		close(chained)
	}()
	return chained
}
