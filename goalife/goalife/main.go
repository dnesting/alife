// An implementation of artificial life.
//
// This binary instantiates a basic world and populates it with
// random organisms whenever the number of organisms drops below a
// certain threshold. A view of the world is rendered to the terminal
// as it evolves.

package main

import "flag"
import "os"
import "sync"
import "time"

import "github.com/dnesting/alife/goalife/world/grid2d"
import "github.com/dnesting/alife/goalife/world/grid2d/term"

var (
	printWorld bool
	printRate  float64
)

func init() {
	flag.BoolVar(&printWorld, "print", true, "render the world to the terminal")
	flag.Float64Var(&printRate, "print_hz", 10.0, "refresh rate in Hz for --print")
	//flag.BoolVar(&debug, "debug", false, "enable tracing")
	//flag.BoolVar(&pprof, "pprof", false, "enable profiling")
}

func main() {
	flag.Parse()
	exit := make(chan bool, 0)

	g := grid2d.New(200, 50)

	if printWorld {
		dur := time.Duration(1.0/printRate) * time.Second
		chooseRune := func(o interface{}) rune {
			return '?'
		}
		go term.Printer(os.Stdout, g, chooseRune, dur, exit)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
}
