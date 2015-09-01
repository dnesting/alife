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
import "math/rand"
import "os"
import "path"
import "sync"
import "time"

import "github.com/dnesting/alife/goalife/entities"
import "github.com/dnesting/alife/goalife/entities/org/cpuorg"
import "github.com/dnesting/alife/goalife/entities/census"
import "github.com/dnesting/alife/goalife/sim"
import "github.com/dnesting/alife/goalife/world"

// syncUpdate synchronizes an organism's execution until its last
// operation gets rendered. This greatly slows execution, but allows
// for a more pleasing visual representation of the organisms as
// they move about. When this is false, a great many movements of the
// organisms can occur between renderings.
const syncUpdate = true

// ensureOrgs is the number of organisms we should attempt to maintain
// in the world at all times. We will populate the world with randomly-
// generated organisms as needed.
const ensureOrgs = 50

// initialEnergy is the energy that should be given to the organisms we
// use to seed the world.
const initialEnergy = 20000

// fractionFromHistory controls how often our randomly-generated organism
// is actually just something we pull out of the census history.
const fractionFromHistory = 0.01

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

func putRandomOrg(s *sim.Sim) {
	o := cpuorg.Random()
	o.AddEnergy(initialEnergy)
	o.PlaceRandomly(s, o)
	s.Start(o)
}

func resurrectOrg(s *sim.Sim) {
	c, err := s.Census.Random()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	if c != nil {
		o := cpuorg.FromCode(c.Genome.Code())
		o.AddEnergy(initialEnergy)
		o.PlaceRandomly(s, o)
		s.Start(o)
	}
}

func ensureMinimumOrgs(s *sim.Sim, count int) {
	for i := count; i < ensureOrgs; i++ {
		if rand.Float32() < fractionFromHistory {
			resurrectOrg(s)
		} else {
			putRandomOrg(s)
		}
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

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
		w = world.New(50, 200)
	}

	// We want to consider food pellets to be equivalent to an empty cell for
	// the purposes of placing a new organism.
	w.ConsiderEmpty(func(o world.Occupant) bool {
		if _, ok := o.(*entities.Food); ok {
			return true
		}
		return false
	})

	// The Sim instance manages most aspects of the simulation.
	s := sim.NewSim(w)
	s.MutateOnDivideProb = 0.01
	s.BodyEnergy = 1000
	s.SenseDistance = 10

	// Use a Census instance to track the evolution of "genomes" over time.
	s.Census = census.NewDirCensus("/tmp/census", recordAtPopulation)
	s.Census.OnChange(func(b census.Census, _ *census.Cohort, _ bool) {
		if !s.IsStopped() {
			ensureMinimumOrgs(s, b.Count())
		}
	})

	// Just for fun
	w.PlaceRandomly(entities.NewFood(1000))
	w.PlaceRandomly(entities.NewFood(1000))
	w.PlaceRandomly(entities.NewFood(1000))
	w.PlaceRandomly(entities.NewFood(1000))

	// Start us off with one organism.  We need to explicitly add one so that
	// the census update gets triggered to get the rest added.
	putRandomOrg(s)

	// Clear the screen
	fmt.Print("\033[H\033[2J")

	// Start rendering updates to the screen periodically.
	screenUpdated, screenTicker := startScreenUpdates(s, &frame, refreshHz)
	defer screenTicker.Stop()

	// Start auto-saving the world periodically.
	autoSaveTicker := startAutoSave(w, &frame, autoSaveSecs)
	defer autoSaveTicker.Stop()

	// This is called every time the world changes somehow.
	w.OnUpdate(func(w world.World) {
		frame += 1

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
	})

	s.Run()
}

// startScreenUpdates begins rendering s.World every 1/refreshHz seconds.
// It returns a sync.Cond instance that gets triggered after every rendering,
// and a time.Ticker instance that can be stopped to halt rendering.
func startScreenUpdates(s *sim.Sim, frame *int64, refreshHz int) (*sync.Cond, *time.Ticker) {
	printed := sync.NewCond(&sync.Mutex{})
	ticker := time.NewTicker(time.Second / time.Duration(refreshHz))

	go func() {
		for range ticker.C {
			fmt.Print("\033[H")
			fmt.Println(s.World)
			fmt.Printf("update %d\n", *frame)
			fmt.Printf("seen %d/%d (%d/%d species, %d recorded)     \n",
				s.Census.Count(), s.Census.CountAllTime(),
				s.Census.Distinct(), s.Census.DistinctAllTime(),
				s.Census.NumRecorded)
			x, y := s.World.Dimensions()
			fmt.Printf("random: %+v\033[K\n",
				s.World.At(rand.Intn(x), rand.Intn(y)))
			if syncUpdate {
				printed.Broadcast()
			}
		}
	}()

	return printed, ticker
}

func registerGobTypes() {
	gob.Register(&entities.Food{})
	gob.Register(&cpuorg.CpuOrganism{})
}

func saveWorld(w world.World, frame *int64) error {
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

func restoreWorld(frame *int64) (world.World, error) {
	f, err := os.Open(path.Join(autoSaveDirectory, autoSaveFilename))
	if err != nil {
		return nil, err
	}

	registerGobTypes()
	dec := gob.NewDecoder(f)
	w := &world.BasicWorld{}
	if err := dec.Decode(w); err != nil {
		return nil, err
	}
	dec.Decode(frame)
	return w, nil
}

// startAutoSave begins auto-saving the state of w every autoSaveSecs.
func startAutoSave(w world.World, frame *int64, autoSaveSecs int) *time.Ticker {
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
