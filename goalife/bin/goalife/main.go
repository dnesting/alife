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
import "math/rand"
import "net/http"
import "os"
import "runtime"
import "sync"
import "sync/atomic"
import "time"
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
import "github.com/dnesting/alife/goalife/util/chanbuf"

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
	// Create a new Census that writes to /tmp/census when a population grows to 40.
	cns, err := census.NewDirCensus("/tmp/census", func(p census.Population) bool { return p.Count > 40 })
	if err != nil {
		fmt.Printf("Error creating census: %v\n", err)
		os.Exit(1)
	}

	// Use human times.
	timeNow := func(interface{}) interface{} { return time.Now() }

	ch := make(chan []grid2d.Update, 0)
	g.Subscribe(ch)

	// Populate the Census with what's already in the world (perhaps restored from an autosave).
	// Assumes nothing in the world is changing yet.
	grid2d.ScanForCensus(cns, g, timeNow, orgHash)

	// Start monitoring for changes
	go grid2d.WatchForCensus(cns, ch, timeNow, orgHash)

	return cns
}

func startAndMaintainOrgs(g grid2d.Grid) {
	// Obtain an initial count before we start anything executing.
	mCount := maintain.Count(g, isOrg)

	ch := make(chan []grid2d.Update, 0)
	g.Subscribe(ch)

	// Start all organisms currently existing in the Grid.  We do this *after*
	// subscribing ch so that we don't end up with a wrong count if any organisms
	// divide or die.
	cpu1.StartAll(g)

	go maintain.Maintain(ch, isOrg, func() { startOrg(g) }, minOrgs, mCount)
}

func startUpdateTracker(g grid2d.Grid, numUpdates *int64) {
	ch := make(chan []grid2d.Update, 0)
	go func() {
		for updates := range ch {
			atomic.AddInt64(numUpdates, int64(len(updates)))
		}
	}()
	g.Subscribe(ch)
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
	// Try to keep rendering smooth.
	runtime.LockOSThread()

	for _ = range ch {
		if clearScreen {
			// TODO(dnesting): use termcap/terminfo to do this more portably.
			fmt.Print("[H")
		}
		term.PrintWorld(os.Stdout, g)
		fmt.Println()
		if clearScreen {
			fmt.Print("[J")
		}

		// Write some summary stats after the rendering.
		fmt.Printf("%d updates\n", atomic.LoadInt64(numUpdates))
		fmt.Printf("%d/%d orgs (%d/%d species, %d recorded)\n", cns.Count(), cns.CountAllTime(), cns.Distinct(), cns.DistinctAllTime(), cns.NumRecorded())
		if loc := g.Get(0, 0); loc != nil {
			fmt.Printf("random: %v\n", loc.Value())
		}

		// If we're running with --sync, signal to any goroutines waiting on a rendering that
		// it's OK for them to continue again.
		if cond != nil {
			cond.Broadcast()
		}
	}
}

func startPrintLoop(g grid2d.Grid, cns *census.DirCensus, cond *sync.Cond, numUpdates *int64, clearScreen bool) {
	// We want to use chanbuf.Tick to ensure renders occur at specific intervals regardless
	// of the rate at which updates arrive.  To prevent the notification channel from backing up
	// and causing deadlock, we buffer using a chanbuf.Trigger (since we don't care about the
	// update messages themselves).  We could have used a time.Tick instead, but we'd need to
	// do something special to ensure this loop exits when the notifications stop.

	// grid notifier -> updateCh -> trigger -> tick -> print world
	//   grid notifier first dispatches an update the moment it occurs
	//   trigger consumes the event and flags that an update occurred
	//   tick fires every freq, halting when trigger reports it's closed
	//   printLoop renders the grid every time tick fires

	updateCh := make(chan []grid2d.Update, 0)
	trigger := chanbuf.Trigger()
	freq := time.Duration(1000000.0/printRate) * time.Microsecond
	go chanbuf.Feed(trigger, grid2d.NotifyToInterface(updateCh))
	tickCh := chanbuf.Tick(trigger, freq, true)
	g.Subscribe(updateCh)

	go printLoop(grid2d.NotifyFromInterface(tickCh), g, cns, cond, numUpdates, clearScreen)
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
	g := grid2d.New(0, 0, cond)
	if saveFile != "" {
		if err := autosave.Restore(saveFile, g); err != nil && !os.IsNotExist(err) {
			fmt.Printf("error restoring from %s: %v\n", saveFile, err)
			os.Exit(1)
		}
	}

	// Force the world to conform to --width and --height.
	g.Resize(width, height, nil)

	// Record the contents of the grid (which may not be empty if restored from autosave)
	// and start monitoring it for changes.
	cns := startCensus(g)

	// Start any organisms that exist in the world (e.g., from autosave) and begin tracking
	// the number of organisms and maintaining a minimum number.
	startAndMaintainOrgs(g)

	if saveFile != "" && saveEvery != 0 {
		// Begin auto-saving the world periodically.
		startAutosave(g, exit)
	}

	// Track the number of updates observed in the world for display during rendering.
	var numUpdates int64
	startUpdateTracker(g, &numUpdates)

	if printWorld {
		// Start rendering the world periodically.
		startPrintLoop(g, cns, cond, &numUpdates, !isTracing())
	}

	// Block until exit, which presently is never.
	<-exit

	// For completeness, stop all of the goroutines waiting on world events.
	g.CloseSubscribers()
}
