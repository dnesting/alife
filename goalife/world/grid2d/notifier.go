package grid2d

import "container/list"
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
	cond sync.Cond
	u    list.List
	mu   sync.Mutex
	subs []chan<- []Update
}

func newNotifier(done <-chan bool) *notifier {
	n := &notifier{
		cond: sync.Cond{L: &sync.Mutex{}},
	}
	go n.run(done)
	return n
}

func (n *notifier) Subscribe(ch chan<- []Update) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.subs = append(n.subs, ch)
}

func (n *notifier) Unsubscribe(ch chan<- []Update) {
	n.mu.Lock()
	defer n.mu.Unlock()
	subs := make([]chan<- []Update, 0)
	for _, c := range n.subs {
		if c != ch {
			subs = append(subs, c)
		}
	}
	n.subs = subs
}

func (n *notifier) RecordAdd(x, y int, value interface{}) {
	n.add([]Update{Update{New: &Point{x, y, value}}})
}

func (n *notifier) RecordRemove(x, y int, value interface{}) {
	n.add([]Update{Update{Old: &Point{x, y, value}}})
}

func (n *notifier) RecordMove(x1, y1, x2, y2 int, value interface{}) {
	n.add([]Update{Update{
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
	n.cond.L.Lock()
	defer n.cond.L.Unlock()
	n.u.PushBack(u)
	n.cond.Signal()
}

func (n *notifier) next() []Update {
	n.cond.L.Lock()
	defer n.cond.L.Unlock()
	var e *list.Element
	for e == nil {
		e = n.u.Front()
		if e == nil {
			n.cond.Wait()
		}
	}
	n.u.Remove(e)
	return e.Value.([]Update)
}

func (n *notifier) run(exitCh <-chan bool) {
	if exitCh != nil {
		go func() {
			<-exitCh
			n.add(nil)
		}()
	}
	for {
		u := n.next()
		if u == nil {
			break
		}
		n.mu.Lock()
		subs := n.subs
		n.mu.Unlock()

		for _, ch := range subs {
			ch <- u
		}
	}
}
