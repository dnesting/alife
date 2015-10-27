// The Grid supports notifications for when changes are made.
// Notifications are delivered via a chan <-[]Update, which, to
// guarantee ordering and maximize performance, are delivered in
// the critical path of the mutation itself.  Subscriber goroutines
// should receive updates promptly to avoid stalling the update,
// and must not directly interact with the world itself to avoid
// deadlocking.  Subscribers can insulate themselves from these
// conditions by queueing updates.  See the util/chanbuf package
// for one method to accomplish this.
package grid2d

import "sync"

// Update represents a notification event of a change occuring to a Grid.
type Update struct {
	Old *Point
	New *Point
}

// IsAdd returns true if the Update represents new occupant added to the Grid.
// u.New will be set to a Point describing the new occupant.
func (u Update) IsAdd() bool {
	return u.Old == nil && u.New != nil
}

// IsRemove returns true if the Update represents an occupant being removed from the Grid.
// u.Old will be set to a Point describing the removed occupant.
func (u Update) IsRemove() bool {
	return u.Old != nil && u.New == nil
}

// IsMove returns true if the Update represents an occupant being moved from one location
// to another.  u.Old and u.New will be set to Points describing the occupant moved.
func (u Update) IsMove() bool {
	return u.Old != nil && u.New != nil && (u.Old.X != u.New.X || u.Old.Y != u.New.Y)
}

// IsReplace returns true if the Update represents an occupant being replaced without
// being moved.  u.Old and u.New will be set to Points describing the occupant change.
func (u Update) IsReplace() bool {
	return u.Old != nil && u.New != nil && u.Old.V != u.New.V
}

type notifier struct {
	mu   sync.Mutex
	subs []chan<- []Update
}

// CloseSubscribers iterates over the notification subscribers and closes them, so as
// to signal consumers that no more notifications will be arriving.  It is illegal to
// continue to mutate a Grid after this method is called.
func (n *notifier) CloseSubscribers() {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, ch := range n.subs {
		close(ch)
	}
}

// Subscribe adds ch to the list of notification subscribers, which will begin receiving
// events immediately as the Grid is mutated.
func (n *notifier) Subscribe(ch chan<- []Update) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.subs = append(n.subs, ch)
}

// Unsubscribe removes ch from the list of notification subscribers.  No further
// notifications will be sent to ch once this method returns.
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

// RecordAdd records an Add notification for the given occupant.
func (n *notifier) RecordAdd(x, y int, value interface{}) {
	n.add([]Update{
		Update{
			New: &Point{x, y, value},
		}})
}

// RecordRemove records a Remove notification for the given occupant.
func (n *notifier) RecordRemove(x, y int, value interface{}) {
	n.add([]Update{
		Update{
			Old: &Point{x, y, value},
		}})
}

// RecordMove records a Move notification for the given occupant.
// x1,y1 represents the original location and x2,y2 represents the new one.
func (n *notifier) RecordMove(x1, y1, x2, y2 int, value interface{}) {
	n.add([]Update{
		Update{
			Old: &Point{x1, y1, value},
			New: &Point{x2, y2, value},
		}})
}

// RecordReplace records a Replace notification for the given occupants.
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

// NotifyToInterface provides a type conversion from <-chan []Update to
// <-chan interface{}.  Go does not provide a facility for this natively,
// so we resort to running a goroutine to shuttle values concurrently.
// This is used to provide compatibility between a Grid's notification
// messages and the queueing and rate-limiting features in util/chanbuf.
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

// NotifyFromInterface provides a type conversion from <-chan []interface{}
// to <-chan []Update.  Go does not provide a facility for this natively,
// so we resort to running a goroutine to shuttle values concurrently.
// This is used to provide compatibility between a Grid's notification
// messages and the queueing and rate-limiting features in util/chanbuf.
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
