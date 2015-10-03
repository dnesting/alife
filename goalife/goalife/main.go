// An implementation of artificial life.
//
// This binary instantiates a basic world and populates it with
// random organisms whenever the number of organisms drops below a
// certain threshold. A view of the world is rendered to the terminal
// as it evolves.

package main

import "encoding/gob"
import "fmt"
import "io/ioutil"
import "log"
import "math/rand"
import "net/http"
import "os"
import "path"
import "sync"
import "sync/atomic"
import "time"
import _ "net/http/pprof"

import "github.com/dnesting/alife/goalife/entities"
import "github.com/dnesting/alife/goalife/entities/org/cpuorg"
import "github.com/dnesting/alife/goalife/entities/census"
import "github.com/dnesting/alife/goalife/sim"
import "github.com/dnesting/alife/goalife/world"
import "github.com/dnesting/alife/goalife/world/text"

const printWorld = true
const tracing = false

// syncUpdate synchronizes an organism's execution until its last
// operation gets rendered. This greatly slows execution, but allows
// for a more pleasing visual representation of the organisms as
// they move about. When this is false, a great many movements of the
// organisms can occur between renderings.
const syncUpdate = false

// refreshHz controls the rate at which we will attempt to re-render
// the world in the terminal.
const refreshHz = 30

// recordAtPopulation will trigger a census recording of any genome that
// reaches this many organisms living at once.
const recordAtPopulation = 40

// autoSaveDirectory is the directory within which we will auto-save the
// world.
const autoSaveDirectory = "/tmp"

// autoSaveFilename is the filename to which we will auto-save the world
// within autoSaveDirectory.
const autoSaveFilename = "autosave.dat"

// autoSaveSecs is how often we will attempt to auto-save the world.
const autoSaveSecs = 1

// pprof determines whether to enable profiling
const pprof = true

func initWorld(w *world.World) {
	// We want to consider food pellets to be equivalent to an empty cell for
	// the purposes of placing a new organism.
	w.EmptyFn = func(o interface{}) bool {
		if _, ok := o.(*entities.Food); ok {
			return true
		}
		return false
	}
}

func createOrg(s *sim.Sim, c census.Census) interface{} {
	d, ok := c.(*census.DirCensus)
	var o *cpuorg.CpuOrganism
	if ok && rand.Float32() < s.FractionFromHistory {
		if cohort, err := d.Random(); err == nil {
			if cohort != nil {
				o = cpuorg.FromCode(cohort.Genome.Code())
			}
		} else {
			fmt.Println(err.Error())
		}
	} else {
		o = cpuorg.Random()
	}
	if o != nil {
		o.AddEnergy(s.InitialEnergy)
		o.PlaceRandomly(s, o)
		return o
	}
	return nil
}

func main() {
	rand.Seed(time.Now().UnixNano())

	if pprof {
		go func() {
			log.Println(http.ListenAndServe("0.0.0.0:6060", nil))
		}()
	}

	// This just accumulates some notion of "progress" in the world, currently
	// by counting the number of world-changing events (e.g., organism movement).
	var frame int64

	// First attempt to restore the world from an auto-save
	w, err := restoreWorld(&frame)
	if err != nil && !os.IsNotExist(err) {
		panic(fmt.Sprintf("%v", err))
	}

	// Otherwise instantiate a new world
	if w == nil {
		w = world.New(200, 50)
		initWorld(w)

		// Just for fun
		w.PlaceRandomly(entities.NewFood(1000))
		w.PlaceRandomly(entities.NewFood(1000))
		w.PlaceRandomly(entities.NewFood(1000))
		w.PlaceRandomly(entities.NewFood(1000))
	}

	// Use a Census instance to track the evolution of "genomes" over time.
	c := census.NewDirCensus("/tmp/census", recordAtPopulation)

	// The Sim instance manages most aspects of the simulation.
	s := sim.NewSim(w, c)
	s.MinimumOrgs = 50
	s.BodyEnergy = 1000
	s.InitialEnergy = 5000
	s.SenseDistance = 10
	s.FractionFromHistory = 0.0001
	s.MutateOnDivideProb = 0.01
	s.OrgFactory = func() interface{} {
		return createOrg(s, c)
	}

	if tracing {
		s.Tracer = os.Stdout
		w.Tracer = os.Stdout
	}

	// Start rendering updates to the screen periodically.
	screenUpdated, screenTicker := startScreenUpdates(s, &frame, refreshHz)
	defer screenTicker.Stop()

	// Start auto-saving the world periodically.
	autoSaveTicker := startAutoSave(w, &frame, autoSaveSecs)
	defer autoSaveTicker.Stop()

	// This is called every time the world changes somehow.
	w.UpdateFn = func(w *world.World) {
		atomic.AddInt64(&frame, 1)

		// If we want synchronous renderings, we just block
		// here until a rendering occurs. This effectively
		// blocks the goroutine that triggered the world
		// update, meaning that organisms that performed a
		// world-changing action won't get to do another
		// one until their last action got rendered.
		if syncUpdate {
			screenUpdated.L.Lock()
			defer screenUpdated.L.Unlock()
			screenUpdated.Wait()
		}
	}

	s.Run()
}

// startScreenUpdates begins rendering s.World every 1/refreshHz seconds.
// It returns a sync.Cond instance that gets triggered after every rendering,
// and a time.Ticker instance that can be stopped to halt rendering.
func startScreenUpdates(s *sim.Sim, frame *int64, refreshHz int) (*sync.Cond, *time.Ticker) {
	if printWorld {
		// Clear the screen
		fmt.Print("\033[H\033[2J")
	}

	printed := sync.NewCond(&sync.Mutex{})
	ticker := time.NewTicker(time.Second / time.Duration(refreshHz))

	go func() {
		for range ticker.C {
			if printWorld {
				if !tracing {
					fmt.Print("\033[H")
				}
				fmt.Println(text.WorldAsString(s.World))
				fmt.Printf("update %d, steps %d\n", *frame, cpuorg.StepCount())
				fmt.Printf("seen %d/%d (%d/%d species",
					s.Census.Count(), s.Census.CountAllTime(),
					s.Census.Distinct(), s.Census.DistinctAllTime())
				if dc, ok := s.Census.(*census.DirCensus); ok {
					fmt.Printf(", %d recorded", dc.NumRecorded)
				}
				fmt.Println("     ")
				x, y := s.World.Width(), s.World.Height()
				fmt.Printf("random: %+v\033[K\n",
					s.World.At(rand.Intn(x), rand.Intn(y)).Value())
			} else if tracing {
				fmt.Println("-- printed --")
			}
			if syncUpdate {
				printed.Broadcast()
			}
		}
	}()

	return printed, ticker
}

func registerGobTypes() {
	gob.Register(&world.Entity{})
	gob.Register(&entities.Food{})
	gob.Register(&cpuorg.CpuOrganism{})
}

func saveWorld(w *world.World, frame *int64) error {
	f, err := ioutil.TempFile(autoSaveDirectory, autoSaveFilename)
	if err != nil {
		return err
	}
	defer f.Close()

	registerGobTypes()
	enc := gob.NewEncoder(f)
	if err := enc.Encode(w); err != nil {
		f.Close()
		os.Remove(f.Name())
		return err
	}
	if err := enc.Encode(frame); err != nil {
		f.Close()
		os.Remove(f.Name())
		return err
	}

	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return err
	}
	if err := os.Rename(f.Name(), path.Join(autoSaveDirectory, autoSaveFilename)); err != nil {
		os.Remove(f.Name())
		return err
	}
	return nil
}

func restoreWorld(frame *int64) (*world.World, error) {
	f, err := os.Open(path.Join(autoSaveDirectory, autoSaveFilename))
	if err != nil {
		return nil, err
	}

	registerGobTypes()
	dec := gob.NewDecoder(f)
	w := &world.World{}
	initWorld(w)
	if err := dec.Decode(w); err != nil {
		return nil, err
	}
	dec.Decode(frame)
	return w, nil
}

// startAutoSave begins auto-saving the state of w every autoSaveSecs.
func startAutoSave(w *world.World, frame *int64, autoSaveSecs int) *time.Ticker {
	ticker := time.NewTicker(time.Duration(autoSaveSecs) * time.Second)

	go func() {
		for range ticker.C {
			if err := saveWorld(w, frame); err != nil {
				fmt.Printf("autosave: %v\n", err)
				continue
			}
		}
	}()

	return ticker
}
