package sim

import "errors"
import "sync"

//import "entities"
import "world"

type Sim struct {
	World world.World
}

type Steppable interface {
	Step(w world.World, x, y int)
}

type StepCallback func(w world.World) error

var StopRunning = errors.New("simulation complete")
var NothingToExecute = errors.New("nothing in the world to execute")

func (s *Sim) Step(cb StepCallback) error {
	var steppers []func()

	s.World.Each(func(x, y int, o world.Occupant) {
		if st, ok := o.(Steppable); ok {
			steppers = append(steppers, func() {
				st.Step(s.World, x, y)
			})
		}
	})

	var wg sync.WaitGroup
	for _, fn := range steppers {
		fn := fn
		wg.Add(1)
		go func() {
			fn()
			wg.Done()
		}()
	}
	wg.Wait()

	if len(steppers) == 0 {
		return NothingToExecute
	}

	return cb(s.World)
}

func (s *Sim) Run(fn StepCallback) error {
	for {
		if e := s.Step(fn); e != nil {
			if e == StopRunning {
				return nil
			}
			return e
		}
	}
}
