package sim

import "sync"
import "time"

import "entities/census"
import "world"

type Sim struct {
	World  world.World
	Census *census.DirCensus

	mu   sync.RWMutex
	wg   sync.WaitGroup
	stop bool
}

func NewSim(w world.World) *Sim {
	return &Sim{
		World: w,
	}
}

func (s *Sim) StopAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stop = true
}

func (s *Sim) IsStopped() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.stop
}

type Runnable interface {
	Run(s *Sim)
}

func (s *Sim) Time() int64 {
	return time.Now().UnixNano()
}

func (s *Sim) Start(st Runnable) {
	s.wg.Add(1)
	if g, ok := st.(census.Genomer); ok {
		s.Census.Add(s.Time(), g.Genome())
	}
	go func() {
		defer s.wg.Done()
		if g, ok := st.(census.Genomer); ok {
			defer func() {
				s.Census.Remove(s.Time(), g.Genome())
			}()
		}
		st.Run(s)
	}()
}

func (s *Sim) Run() {
	s.stop = false
	s.World.Each(func(x, y int, o world.Occupant) {
		if st, ok := o.(Runnable); ok {
			s.Start(st)
		}
	})
	s.wg.Wait()
}
