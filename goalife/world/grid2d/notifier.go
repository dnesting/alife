package grid2d

import "fmt"
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

type NotifyQueue interface {
	Next() []Update
	Add(u []Update)
	Done()
}

type BufferedNotifyQueue struct {
	style NotifyStyle
	c     *sync.Cond
	u     []Update
	stop  bool
}

func (q *BufferedNotifyQueue) Next() []Update {
	q.c.L.Lock()
	defer q.c.L.Unlock()
	for q.u == nil && !q.stop {
		q.c.Wait()
	}
	if q.stop {
		return nil
	}
	u := q.u
	q.u = nil
	return u
}

func (q *BufferedNotifyQueue) Add(u []Update) {
	q.c.L.Lock()
	defer q.c.L.Unlock()
	switch q.style & (BufferFirst | BufferLast | BufferAll) {
	case BufferFirst:
		if q.u == nil {
			q.u = u
		}
	case BufferLast:
		q.u = u
	case BufferAll:
		q.u = append(q.u, u...)
	default:
		panic(fmt.Sprintf("illegal queue style: %v", q.style))
	}
	q.c.Signal()
}

func (q *BufferedNotifyQueue) Done() {
	q.c.L.Lock()
	defer q.c.L.Unlock()
	q.stop = true
	q.c.Signal()
}

type UnbufferedNotifyQueue chan []Update

func (q UnbufferedNotifyQueue) Add(u []Update) {
	q <- u
}

func (q UnbufferedNotifyQueue) Next() []Update {
	return <-q
}

func (q UnbufferedNotifyQueue) Done() {
	close(q)
}

func NewNotifyQueue(style NotifyStyle) NotifyQueue {
	if style&Unbuffered != 0 {
		return UnbufferedNotifyQueue(make(chan []Update, 0))
	} else {
		return &BufferedNotifyQueue{
			style: style,
			c:     sync.NewCond(&sync.Mutex{}),
		}
	}
}

func notifyFromQueue(ch chan<- []Update, queue NotifyQueue) {
	for u := queue.Next(); u != nil; u = queue.Next() {
		ch <- u
	}
	close(ch)
}

type notifier struct {
	cond sync.Cond
	u    list.List
	mu   sync.Mutex
	subs map[chan<- []Update]NotifyQueue
}

func newNotifier(done <-chan bool) *notifier {
	n := &notifier{
		cond: sync.Cond{L: &sync.Mutex{}},
		subs: make(map[chan<- []Update]NotifyQueue),
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
	for _, q := range n.subs {
		q.Done()
	}
}

func (n *notifier) Subscribe(ch chan<- []Update, style NotifyStyle) {
	n.mu.Lock()
	defer n.mu.Unlock()
	q := NewNotifyQueue(style)
	n.subs[ch] = q
	go notifyFromQueue(ch, q)
}

func (n *notifier) Unsubscribe(ch chan<- []Update) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if q, ok := n.subs[ch]; ok {
		q.Done()
		delete(n.subs, ch)
	}
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
	for _, q := range n.subs {
		q.Add(u)
	}
}
