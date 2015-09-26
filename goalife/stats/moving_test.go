package stats

import "testing"
import "time"

type fakeClock struct {
	T time.Time
}

func (f *fakeClock) Advance(d time.Duration) time.Time {
	f.T = f.T.Add(d)
	return f.Now()
}

func (f *fakeClock) Now() time.Time {
	return f.T
}

func TestMovingAvg(t *testing.T) {
	avg := MovingAvg(3 * time.Second)
	fc := &fakeClock{time.Now()}
	var oldClock clock
	oldClock, clk = clk, fc
	defer func() { clk = oldClock }()

	if avg.Valid() {
		t.Fatal("initial MovingAvg is valid but shouldn't be")
	}
	avg.Value() // shouldn't panic

	avg.Add(1.0)
	if avg.Value() != 1.0 {
		t.Errorf("[1.0] should average 1.0, got %.2f", avg.Value())
	}
	fc.Advance(1 * time.Second)

	avg.Add(2.0)
	if avg.Value() != 1.5 {
		t.Errorf("[1.0 2.0] should average 1.5, got %.2f", avg.Value())
	}
	fc.Advance(1 * time.Second)

	avg.Add(3.0)
	if avg.Value() != 2.0 {
		t.Errorf("[1.0 2.0 3.0] should average 2.0, got %.2f", avg.Value())
	}
	fc.Advance(1 * time.Second)

	avg.Add(4.0)
	if avg.Value() != 3.0 {
		t.Errorf("[x1.0 2.0 3.0 4.0] should average 3.0, got %.2f", avg.Value())
	}
	fc.Advance(1 * time.Second)

	avg.Add(5.0)
	if avg.Value() != 4.0 {
		t.Errorf("[x1.0 x2.0 3.0 4.0 5.0] should average 4.0, got %.2f", avg.Value())
	}
}
