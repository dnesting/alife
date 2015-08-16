package sim

import "fmt"
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

func (s *Sim) Step(fn StepCallback) error {
	var wg sync.WaitGroup
	var i int

	w := s.World.Copy()

	w.Each(func(x, y int, o world.Occupant) {
		fmt.Printf("got (%d,%d) %v\n", x, y, o)
		if st, ok := o.(Steppable); ok {
			fmt.Println("steppable")
			i += 1
			wg.Add(1)
			go func() {
				st.Step(w, x, y)
				fmt.Println("stepped")
				wg.Done()
			}()
		}
	})
	wg.Wait()
	s.World = w

	if i == 0 {
		fmt.Println("nothing to do")
		return NothingToExecute
	}

	return fn(s.World)
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
