// An implementation of artificial life.
//
// This binary instantiates a basic world and populates it with
// random organisms whenever the number of organisms drops below a
// certain threshold. A view of the world is rendered to the terminal
// as it evolves.

package main

import "encoding/gob"
import "flag"
import "fmt"
import "os"
import "math/rand"
import "runtime"
import "sync"
import "sync/atomic"
import "time"
import "net/http"
import _ "net/http/pprof"

import "github.com/dnesting/alife/goalife/census"
import "github.com/dnesting/alife/goalife/grid2d"
import "github.com/dnesting/alife/goalife/grid2d/autosave"
import "github.com/dnesting/alife/goalife/grid2d/food"

import "github.com/dnesting/alife/goalife/grid2d/maintain"
import "github.com/dnesting/alife/goalife/grid2d/org"
import "github.com/dnesting/alife/goalife/grid2d/org/cpu1"
import "github.com/dnesting/alife/goalife/log"
import "github.com/dnesting/alife/goalife/term"

var Logger = log.Null()

const initialEnergy = 10000

var (
	printWorld    bool
	printRate     float64
	pprof         bool
	minOrgs       int
	syncToRender  bool
	saveFile      string
	saveEvery     int
	width, height int

	traceAll      bool
	traceCpu      bool
	traceGrid     bool
	traceMaintain bool
	traceOrg      bool
)

func init() {
	flag.BoolVar(&printWorld, "print", true, "render the world to the terminal")
	flag.Float64Var(&printRate, "print_hz", 10.0, "refresh rate in Hz for --print")
	flag.BoolVar(&pprof, "pprof", false, "enable profiling")
	flag.IntVar(&minOrgs, "min", 50, "maintain this many organisms at a minimum")
	flag.BoolVar(&syncToRender, "sync", false, "sync world updates to rendering")
	flag.StringVar(&saveFile, "save-file", "/tmp/autosave.dat", "auto-save to this filename")
	flag.IntVar(&saveEvery, "save-every", 3, "auto-save every save-every secs")
	flag.IntVar(&width, "width", 200, "width of world")
	flag.IntVar(&height, "height", 50, "height of world")

	flag.BoolVar(&traceAll, "trace-all", false, "enable all tracing")
	flag.BoolVar(&traceCpu, "trace-cpu", false, "enable cpu tracing")
	flag.BoolVar(&traceGrid, "trace-grid", false, "enable grid tracing")
	flag.BoolVar(&traceMaintain, "trace-maintain", false, "enable maintain tracing")
	flag.BoolVar(&traceOrg, "trace-org", false, "enable org tracing")
}

func startOrg(g grid2d.Grid) {
	c := cpu1.Random()
	o := org.Random()
	o.Driver = c
	o.AddEnergy(initialEnergy)
	for {
		// PutRandomly might fail if there's no room, so just keep trying.
		if _, loc := g.PutRandomly(o, org.PutWhenFood); loc != nil {
			go c.Run(o)
			break
		}
	}
}

func startAll(g grid2d.Grid) {
	var locs []grid2d.Point
	g.Locations(&locs)
	for _, p := range locs {
		if o, ok := p.V.(*org.Organism); ok {
			if c, ok := o.Driver.(*cpu1.Cpu); ok {
				go c.Run(o)
			}
		}
	}
}

func isOrg(o interface{}) bool {
	_, ok := o.(*org.Organism)
	return ok
}

func orgHash(o interface{}) *census.Key {
	if o, ok := o.(*org.Organism); ok {
		if c, ok := o.Driver.(*cpu1.Cpu); ok {
			i := census.Key(c)
			return &i
		}
	}
	return nil
}

func setupTracing() {
	l := log.Real()
	if traceAll || traceCpu {
		cpu1.Logger = l
	}
	if traceAll || traceGrid {
		grid2d.Logger = l
	}
	if traceAll || traceMaintain {
		maintain.Logger = l
	}
	if traceAll || traceOrg {
		org.Logger = l
	}
}

func setupPprof() {
	runtime.SetBlockProfileRate(1000)
	go func() {
		Logger.Println(http.ListenAndServe("0.0.0.0:6060", nil))
	}()
}

func registerGob() {
	gob.Register(time.Time{})
	gob.Register(&cpu1.Cpu{})
	gob.Register(&food.Food{})
	gob.Register(&org.Organism{})
}

func startCensus(g grid2d.Grid) *census.DirCensus {
	ch := make(chan []grid2d.Update, 0)
	cns, err := census.NewDirCensus("/tmp/census", func(p census.Population) bool { return p.Count > 30 })
	if err != nil {
		fmt.Printf("Error creating census: %v\n", err)
		os.Exit(1)
	}
	timeNow := func() interface{} { return time.Now() }
	g.Subscribe(ch, grid2d.Unbuffered)
	grid2d.ScanForCensus(cns, g, timeNow, orgHash)
	go grid2d.WatchForCensus(cns, g, ch, timeNow, orgHash)
	return cns
}

func startAndMaintainOrgs(g grid2d.Grid) {
	ch := make(chan []grid2d.Update, 0)
	mCount := maintain.Count(g, isOrg)
	g.Subscribe(ch, grid2d.Unbuffered)
	startAll(g)

	go maintain.Maintain(g, ch, isOrg, func() { startOrg(g) }, minOrgs, mCount)
}

func startUpdateTracker(g grid2d.Grid, numUpdates *int64) {
	ch := make(chan []grid2d.Update, 0)
	go func() {
		for updates := range ch {
			atomic.AddInt64(numUpdates, int64(len(updates)))
		}
	}()
	g.Subscribe(ch, grid2d.Unbuffered)
}

func startAutosave(g grid2d.Grid, exit <-chan bool) {
	go func() {
		err := autosave.Loop(saveFile, g, time.Duration(saveEvery)*time.Second, exit)
		if err != nil {
			fmt.Printf("autosave: %v\n", err)
			os.Exit(1)
		}
	}()
}

func printLoop(ch <-chan []grid2d.Update, g grid2d.Grid, cns *census.DirCensus, cond *sync.Cond, numUpdates *int64, clearScreen bool) {
	runtime.LockOSThread()
	for _ = range ch {
		if clearScreen {
			fmt.Print("[H")
		}
		term.PrintWorld(os.Stdout, g)
		fmt.Println()
		if clearScreen {
			fmt.Print("[J")
		}
		fmt.Printf("%d updates\n", atomic.LoadInt64(numUpdates))
		fmt.Printf("%d/%d orgs (%d/%d species, %d recorded)\n", cns.Count(), cns.CountAllTime(), cns.Distinct(), cns.DistinctAllTime(), cns.NumRecorded())
		if loc := g.Get(0, 0); loc != nil {
			fmt.Printf("random: %v\n", loc.Value())
		}
		if cond != nil {
			cond.Broadcast()
		}
	}
}

func startPrintLoop(g grid2d.Grid, cns *census.DirCensus, cond *sync.Cond, numUpdates *int64, clearScreen bool) {
	freq := time.Duration(1000000.0/printRate) * time.Microsecond
	ch := make(chan []grid2d.Update, 0)
	g.Subscribe(ch, grid2d.BufferLast)
	rateCh := grid2d.RateLimited(ch, freq, 0, true)

	go printLoop(rateCh, g, cns, cond, numUpdates, clearScreen)
}

func isTracing() bool {
	return traceAll || traceGrid || traceOrg || traceMaintain || traceCpu
}

func main() {
	rand.Seed(time.Now().UnixNano())
	flag.Parse()

	setupTracing()
	if pprof {
		setupPprof()
	}

	exit := make(chan bool, 0)

	var cond *sync.Cond
	if syncToRender {
		cond = sync.NewCond(&sync.Mutex{})
	}

	registerGob()

	// Set up the Grid, and restore it from autosave if able.
	g := grid2d.New(0, 0, exit, cond)
	if saveFile != "" {
		if err := autosave.Restore(saveFile, g); err != nil && !os.IsNotExist(err) {
			fmt.Printf("error restoring from %s: %v\n", saveFile, err)
			os.Exit(1)
		}
	}
	g.Resize(width, height, nil)

	cns := startCensus(g)

	// Start any organisms that exist in the world (from autosave) and begin tracking
	// the number of organisms and maintaining a minimum number.
	startAndMaintainOrgs(g)

	if saveFile != "" && saveEvery != 0 {
		startAutosave(g, exit)
	}

	var numUpdates int64
	startUpdateTracker(g, &numUpdates)

	if printWorld {
		startPrintLoop(g, cns, cond, &numUpdates, !isTracing())
	}

	<-exit
}
