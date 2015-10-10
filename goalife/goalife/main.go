// An implementation of artificial life.
//
// This binary instantiates a basic world and populates it with
// random organisms whenever the number of organisms drops below a
// certain threshold. A view of the world is rendered to the terminal
// as it evolves.

package main

import "flag"
import "log"
import "os"
import "sync"
import "time"
import "net/http"
import _ "net/http/pprof"

import "github.com/dnesting/alife/goalife/census"
import "github.com/dnesting/alife/goalife/driver/cpu1"
import "github.com/dnesting/alife/goalife/energy"

import "github.com/dnesting/alife/goalife/maintain"
import "github.com/dnesting/alife/goalife/org"
import "github.com/dnesting/alife/goalife/term"
import "github.com/dnesting/alife/goalife/world/grid2d"

var (
	debug      bool
	printWorld bool
	printRate  float64
	pprof      bool
)

func init() {
	flag.BoolVar(&printWorld, "print", true, "render the world to the terminal")
	flag.Float64Var(&printRate, "print_hz", 10.0, "refresh rate in Hz for --print")
	flag.BoolVar(&debug, "debug", false, "enable tracing")
	flag.BoolVar(&pprof, "pprof", false, "enable profiling")
}

func startOrg(g grid2d.Grid) {
	c := cpu1.Random()
	o := &org.Organism{Driver: c}
	o.AddEnergy(1000)
	if _, loc := g.PutRandomly(o, org.PutWhenFood); loc != nil {
		go c.Run(o)
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
	if pprof {
		go func() {
			log.Println(http.ListenAndServe("127.0.0.1:6060", nil))
		}()
	}
	if debug {
		l := log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile)
		cpu1.Logger = l
		org.Logger = l
		grid2d.Logger = l
		maintain.Logger = l
	}

	exit := make(chan bool, 0)

	g := grid2d.New(200, 50, exit)

	var ch chan []grid2d.Update
	ch = make(chan []grid2d.Update, 0)
	g.Subscribe(ch)
	cns := census.NewDirCensus("/tmp/census", func(p census.Population) bool { return p.Count > 30 })
	go census.WatchWorld(cns, ch, func() interface{} { return time.Now() }, orgHash)

	ch = make(chan []grid2d.Update, 0)
	g.Subscribe(ch)
	go maintain.Maintain(ch, isOrg, func() { startOrg(g) }, 10)

	g.Put(10, 10, energy.NewFood(10), grid2d.PutAlways)
	g.Put(11, 11, energy.NewFood(2000), grid2d.PutAlways)
	g.Put(12, 12, energy.NewFood(3000), grid2d.PutAlways)
	g.Put(13, 13, energy.NewFood(8000), grid2d.PutAlways)

	var wg sync.WaitGroup
	wg.Add(1)

	if printWorld {
		dur := time.Duration(1.0/printRate) * time.Second
		wg.Add(1)
		go func() {
			term.Printer(os.Stdout, g, nil, !debug, dur)
			wg.Done()
		}()
	}

	wg.Wait()
}
