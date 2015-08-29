package main

import "encoding/gob"
import "fmt"
import "io/ioutil"
import "os"
import "path"

import "entities"
import "entities/org/cpuorg"
import "entities/census"
import "math/rand"
import "sim"
import "sync"
import "time"
import "world"

const syncUpdate = true
const ensureOrgs = 50
const refreshHz = 30
const recordAtPopulation = 40
const initialEnergy = 20000
const autoSaveDirectory = "/tmp"
const autoSaveFilename = "autosave.dat"
const autoSaveSecs = 1
const fractionFromHistory = 0.0

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
	o := cpuorg.FromCode(c.Genome.Code())
	o.AddEnergy(initialEnergy)
	o.PlaceRandomly(s, o)
	s.Start(o)
}

func getProgram() []string {
	if rand.Float32() < 0.5 {
		return []string{
			"L1",
			"Zero",
			"Shl1",
			"Shl1",
			"Shl1",
			"Shl1",
			"Shl1",
			"SwapAC",
			"L2",
			"Forward",
			"Eat",
			"Eat",
			"Eat",
			"Eat",
			"IfLoop",
			"JumpR2",
			"Left",
			"Divide",
			"Right",
			"JumpR1",
		}
	} else {
		return []string{
			"L1",
			"Zero",
			"Shl1",
			"Shl1",
			"Shl1",
			"Shl1",
			"SwapAC",

			"L2",
			"Sense",
			"IfZ",
			"Jump3",
			"Jump4",

			"L3",
			"SwapAC",
			"IfZ",
			"JumpR1",
			"Dec",
			"SwapAC",

			"Left",
			"JumpR2",

			"L4",
			"Forward",
			"Forward",
			"Forward",
			"Eat",
			"Eat",
			"Left",
			"Divide",

			"JumpR1",
		}
	}
}

func putOrg(s *sim.Sim) {
	o := cpuorg.FromCode(getProgram())
	o.AddEnergy(initialEnergy)
	o.Mutate()
	o.PlaceRandomly(s, o)
	s.Start(o)
}

func main() {
	rand.Seed(time.Now().UnixNano())
	var frame int64

	w, err := restoreWorld(&frame)
	if err != nil {
		panic(err.Error())
		fmt.Printf("restore: %v\n", err)
	}
	if w == nil {
		w = world.New(50, 200)
	}

	w.ConsiderEmpty(func(o world.Occupant) bool {
		if _, ok := o.(*entities.Food); ok {
			return true
		}
		return false
	})

	s := sim.NewSim(w)
	s.Census = census.NewDirCensus("/tmp/census", recordAtPopulation)
	s.Census.OnChange(func(b census.Census, _ *census.Cohort, _ bool) {
		if b.Count() < ensureOrgs {
			putRandomOrg(s)
		}
	})

	w.PlaceRandomly(entities.NewFood(1000))
	w.PlaceRandomly(entities.NewFood(1000))
	w.PlaceRandomly(entities.NewFood(1000))
	w.PlaceRandomly(entities.NewFood(1000))
	//putOrg(s)
	putRandomOrg(s)

	fmt.Print("\033[H\033[2J")

	screenUpdated, screenTicker := startScreenUpdates(s, &frame)
	defer screenTicker.Stop()
	autoSaveTicker := startAutoSave(w, &frame)
	defer autoSaveTicker.Stop()

	w.OnUpdate(func(w world.World) {
		frame += 1
		if syncUpdate {
			screenUpdated.L.Lock()
			defer screenUpdated.L.Unlock()
			screenUpdated.Wait()
		}
	})

	s.Run()
}

func startScreenUpdates(s *sim.Sim, frame *int64) (*sync.Cond, *time.Ticker) {
	printed := sync.NewCond(&sync.Mutex{})
	ticker := time.NewTicker(time.Second / refreshHz)

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

func startAutoSave(w world.World, frame *int64) *time.Ticker {
	ticker := time.NewTicker(autoSaveSecs * time.Second)

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
