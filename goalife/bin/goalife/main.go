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
import "github.com/dnesting/alife/goalife/grid2d/org/driver/cpu1"
import "github.com/dnesting/alife/goalife/log"
import "github.com/dnesting/alife/goalife/term"

var Logger = log.Null()

var (
	debug         bool
	printWorld    bool
	printRate     float64
	pprof         bool
	minOrgs       int
	syncToRender  bool
	saveFile      string
	saveEvery     int
	width, height int
)

func init() {
	flag.BoolVar(&printWorld, "print", true, "render the world to the terminal")
	flag.Float64Var(&printRate, "print_hz", 10.0, "refresh rate in Hz for --print")
	flag.BoolVar(&debug, "debug", false, "enable tracing")
	flag.BoolVar(&pprof, "pprof", false, "enable profiling")
	flag.IntVar(&minOrgs, "min", 50, "maintain this many organisms at a minimum")
	flag.BoolVar(&syncToRender, "sync", false, "sync world updates to rendering")
	flag.StringVar(&saveFile, "save_file", "/tmp/autosave.dat", "auto-save to this filename")
	flag.IntVar(&saveEvery, "save_every_secs", 3, "auto-save every save_every_secs secs")
	flag.IntVar(&width, "width", 200, "width of world")
	flag.IntVar(&height, "height", 50, "height of world")
}

func startOrg(g grid2d.Grid) {
	c := cpu1.Random()
	o := org.Random()
	o.Driver = c
	o.AddEnergy(1000)
	for {
		if _, loc := g.PutRandomly(o, org.PutWhenFood); loc != nil {
			go func() {
				//g.Wait()
				c.Run(o)
				//if err := c.Run(o); err != nil {
				//	Logger.Printf("org exited: %v\n", err)
				//}
			}()
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

func main() {
	rand.Seed(time.Now().UnixNano())
	flag.Parse()
	if debug {
		l := log.Real()
		Logger = l
		//cpu1.Logger = l
		org.Logger = l
		//grid2d.Logger = l
		//maintain.Logger = l
	}
	if pprof {
		runtime.SetBlockProfileRate(1000)
		go func() {
			Logger.Println(http.ListenAndServe("0.0.0.0:6060", nil))
		}()
	}

	exit := make(chan bool, 0)

	var cond *sync.Cond
	if syncToRender {
		cond = sync.NewCond(&sync.Mutex{})
	}

	census.RegisterGobTypes()
	gob.Register(&org.Organism{})

	g := grid2d.New(0, 0, exit, cond)

	if saveFile != "" {
		if err := autosave.Restore(saveFile, g); err != nil && !os.IsNotExist(err) {
			fmt.Printf("error restoring from %s: %v\n", saveFile, err)
			os.Exit(1)
		}
	}
	g.Resize(width, height, nil)

	var ch chan []grid2d.Update
	ch = make(chan []grid2d.Update, 0)
	g.Subscribe(ch, grid2d.Unbuffered)
	cns, err := census.NewDirCensus("/tmp/census", func(p census.Population) bool { return p.Count > 30 })
	if err != nil {
		fmt.Printf("Error creating census: %v\n", err)
		os.Exit(1)
	}
	go census.WatchWorld(cns, g, ch, func() interface{} { return time.Now() }, orgHash)

	startAll(g)

	ch = make(chan []grid2d.Update, 0)
	mCount := maintain.Count(g, isOrg)
	g.Subscribe(ch, grid2d.Unbuffered)
	go maintain.Maintain(g, ch, isOrg, func() { startOrg(g) }, minOrgs, mCount)

	var numUpdates int64

	ch = make(chan []grid2d.Update, 0)
	g.Subscribe(ch, grid2d.Unbuffered)
	go func() {
		for updates := range ch {
			atomic.AddInt64(&numUpdates, int64(len(updates)))
		}
	}()

	if saveFile != "" && saveEvery != 0 {
		go func() {
			err := autosave.Loop(saveFile, g, time.Duration(saveEvery)*time.Second, exit)
			if err != nil {
				fmt.Printf("autosave: %v\n", err)
				os.Exit(1)
			}
		}()
	}

	g.Put(10, 10, food.New(10), grid2d.PutAlways)
	g.Put(11, 11, food.New(2000), grid2d.PutAlways)
	g.Put(12, 12, food.New(3000), grid2d.PutAlways)
	g.Put(13, 13, food.New(8000), grid2d.PutAlways)

	var wg sync.WaitGroup
	wg.Add(1)

	if printWorld {
		freq := time.Duration(1000000.0/printRate) * time.Microsecond
		ch = make(chan []grid2d.Update, 0)
		g.Subscribe(ch, grid2d.BufferLast)
		ch := grid2d.RateLimited(ch, freq, 0)

		go func() {
			runtime.LockOSThread()
			for _ = range ch {
				if !debug {
					fmt.Print("[H")
				}
				term.PrintWorld(os.Stdout, g)
				fmt.Println()
				if !debug {
					fmt.Print("[J")
				}
				fmt.Printf("%d updates\n", atomic.LoadInt64(&numUpdates))
				fmt.Printf("%d/%d orgs (%d/%d species, %d recorded)\n", cns.Count(), cns.CountAllTime(), cns.Distinct(), cns.DistinctAllTime(), cns.NumRecorded())
				if cond != nil {
					cond.Broadcast()
				}
			}
		}()
	}

	wg.Wait()
}
