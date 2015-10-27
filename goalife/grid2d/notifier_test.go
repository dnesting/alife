package grid2d

import "reflect"
import "testing"

func TestNotify(t *testing.T) {
	tAdd := []Update{
		Update{
			New: &Point{1, 2, 10},
		},
	}

	tRemove := []Update{
		Update{
			Old: &Point{1, 2, 10},
		},
	}

	tMove := []Update{
		Update{
			Old: &Point{1, 1, 10},
			New: &Point{2, 2, 10},
		},
	}

	tReplace := []Update{
		Update{
			Old: &Point{1, 1, 10},
			New: &Point{1, 1, 11},
		},
	}

	var n notifier
	defer n.CloseSubscribers()

	go func() {
		n.RecordAdd(1, 2, 10)
		n.RecordRemove(1, 2, 10)
		n.RecordMove(1, 1, 2, 2, 10)
		n.RecordReplace(1, 1, 10, 11)
	}()

	ch := make(chan []Update)
	n.Subscribe(ch)

	got := <-ch
	if !reflect.DeepEqual(got, tAdd) {
		t.Errorf("notification failed, expected %v got %v", tAdd, got)
	}
	got = <-ch
	if !reflect.DeepEqual(got, tRemove) {
		t.Errorf("notification failed, expected %v got %v", tRemove, got)
	}
	got = <-ch
	if !reflect.DeepEqual(got, tMove) {
		t.Errorf("notification failed, expected %v got %v", tMove, got)
	}
	got = <-ch
	if !reflect.DeepEqual(got, tReplace) {
		t.Errorf("notification failed, expected %v got %v", tReplace, got)
	}
}
