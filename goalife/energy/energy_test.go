package energy

import "testing"

func TestEnergy(t *testing.T) {
	e := Store{10}

	amt, value := e.AddEnergy(5)
	if amt != 5 {
		t.Errorf("5 added, expected 5, got %d", amt)
	}
	if value != 15 {
		t.Errorf("10+5 should be 15, got %d", value)
	}

	amt, value = e.AddEnergy(-15)
	if amt != -15 {
		t.Errorf("-15 added, expected -15, got %d", amt)
	}
	if value != 0 {
		t.Errorf("15-15 should be 0, got %d", value)
	}

	amt, value = e.AddEnergy(-10)
	if amt != 0 {
		t.Errorf("-10 added from 0, expected 0 actually added, got %d", amt)
	}
	if value != 0 {
		t.Errorf("-10 added from 0, expected value of 0, got %d", value)
	}

	amt, value = e.AddEnergy(5)
	if amt != 5 {
		t.Errorf("5 added, expected 5, got %d", amt)
	}
	if value != 5 {
		t.Errorf("0+5 should be 5, got %d", value)
	}

	amt, value = e.AddEnergy(-10)
	if amt != -5 {
		t.Errorf("-10 added from 5 added, expected -5, got %d", amt)
	}
	if value != 0 {
		t.Errorf("5-10 should be 0, got %d", value)
	}
}
