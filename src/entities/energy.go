package entities

type Energetic interface {
	Energy() int
	AddEnergy(amt int) (int, int)
}

type energy struct {
	v int
}

func Energy(v int) Energetic {
	if v < 0 {
		v = 0
	}
	return &energy{v}
}

func (e *energy) Energy() int {
	return e.v
}

func (e *energy) AddEnergy(amt int) (int, int) {
	v := e.v + amt
	nv := v
	if nv < 0 {
		nv = 0
		amt -= v
	}
	e.v = nv
	return amt, e.v
}
