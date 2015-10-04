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

import "github.com/dnesting/alife/goalife/world/grid2d"

var (
	printWorld bool
	printRate  float32
)

func init() {
	flag.BoolVar(&printWorld, "print", true, "render the world to the terminal")
	flag.IntVar(&printRate, "print_hz", 10.0, "refresh rate in Hz for --print")
	//flag.BoolVar(&debug, "debug", false, "enable tracing")
	//flag.BoolVar(&pprof, "pprof", false, "enable profiling")
}

func main() {
	flag.Parse()

	w := grid2d.New(200, 50)
	if printWorld {
		go grid2d.Printer(os.Stdout, w, printRate)
	}
}
