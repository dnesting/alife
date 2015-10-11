// An implementation of artificial life.
//
// This binary instantiates a basic world and populates it with
// random organisms whenever the number of organisms drops below a
// certain threshold. A view of the world is rendered to the terminal
// as it evolves.

package main

import "flag"
import "fmt"
import "os"
import "sync"
import "sync/atomic"
import "time"
import "net/http"
import _ "net/http/pprof"

import "github.com/dnesting/alife/goalife/census"
import "github.com/dnesting/alife/goalife/driver/cpu1"
import "github.com/dnesting/alife/goalife/energy"

import "github.com/dnesting/alife/goalife/maintain"
import "github.com/dnesting/alife/goalife/log"
import "github.com/dnesting/alife/goalife/org"
import "github.com/dnesting/alife/goalife/term"
import "github.com/dnesting/alife/goalife/world/grid2d"

var Logger = log.Null()

var (
	debug        bool
	printWorld   bool
	printRate    float64
	pprof        bool
	minOrgs      int
	syncToRender bool
)

func init() {
	flag.BoolVar(&printWorld, "print", true, "render the world to the terminal")
	flag.Float64Var(&printRate, "print_hz", 10.0, "refresh rate in Hz for --print")
	flag.BoolVar(&debug, "debug", false, "enable tracing")
	flag.BoolVar(&pprof, "pprof", false, "enable profiling")
	flag.IntVar(&minOrgs, "min", 50, "maintain this many organisms at a minimum")
	flag.BoolVar(&syncToRender, "sync", false, "sync world updates to rendering")
}

func startOrg(g grid2d.Grid) {
	c := cpu1.Random()
	o := &org.Organism{Driver: c}
	o.AddEnergy(1000)
	for {
		if _, loc := g.PutRandomly(o, org.PutWhenFood); loc != nil {
			go func() {
				g.Wait()
				c.Run(o)
			}()
			// go func() {
			// 	if err := c.Run(o); err != nil {
			// 		Logger.Printf("org exited: %v\n", err)
			// 	}
			// }()
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

func main() {
	flag.Parse()
	if debug {
		l := log.Real()
		Logger = l
		//cpu1.Logger = l
		//org.Logger = l
		//grid2d.Logger = l
		maintain.Logger = l
	}
	if pprof {
		go func() {
			Logger.Println(http.ListenAndServe("0.0.0.0:6060", nil))
		}()
	}

	exit := make(chan bool, 0)

	var cond *sync.Cond
	if syncToRender {
		cond = sync.NewCond(&sync.Mutex{})
	}

	g := grid2d.New(200, 50, exit, cond)

	var ch chan []grid2d.Update
	ch = make(chan []grid2d.Update, 0)
	g.Subscribe(ch, grid2d.Unbuffered)
	cns := census.NewDirCensus("/tmp/census", func(p census.Population) bool { return p.Count > 30 })
	go census.WatchWorld(cns, ch, func() interface{} { return time.Now() }, orgHash)

	ch = make(chan []grid2d.Update, 0)
	g.Subscribe(ch, grid2d.Unbuffered)
	go maintain.Maintain(ch, isOrg, func() { startOrg(g) }, minOrgs)

	var numUpdates int64

	ch = make(chan []grid2d.Update, 0)
	g.Subscribe(ch, grid2d.Unbuffered)
	go func() {
		for updates := range ch {
			atomic.AddInt64(&numUpdates, int64(len(updates)))
		}
	}()
	g.Put(10, 10, energy.NewFood(10), grid2d.PutAlways)
	g.Put(11, 11, energy.NewFood(2000), grid2d.PutAlways)
	g.Put(12, 12, energy.NewFood(3000), grid2d.PutAlways)
	g.Put(13, 13, energy.NewFood(8000), grid2d.PutAlways)

	var wg sync.WaitGroup
	wg.Add(1)

	if printWorld {
		freq := time.Duration(1000000.0/printRate) * time.Microsecond
		ch = make(chan []grid2d.Update, 0)
		g.Subscribe(ch, grid2d.BufferLast)
		ch := grid2d.RateLimited(ch, freq, 0)

		go func() {
			for _ = range ch {
				if !debug {
					fmt.Print("[H")
				}
				term.PrintWorld(os.Stdout, g)
				fmt.Println()
				fmt.Printf("%d updates\n", atomic.LoadInt64(&numUpdates))
				fmt.Printf("%d/%d orgs (%d/%d species)[J\n", cns.Count(), cns.CountAllTime(), cns.Distinct(), cns.DistinctAllTime())
				if cond != nil {
					cond.Broadcast()
				}
			}
		}()
	}

	wg.Wait()
}
