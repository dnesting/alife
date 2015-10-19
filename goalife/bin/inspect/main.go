package main

import "encoding/gob"
import "fmt"
import "os"
import "path"
import "time"

import "github.com/dnesting/alife/goalife/census"
import "github.com/dnesting/alife/goalife/grid2d/org/cpu1"

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s /path/to/census-file\n", path.Base(os.Args[0]))
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Printf("%s: %v\n", os.Args[1], err)
		return
	}
	fmt.Printf("reading from %#v\n", *f)
	dec := gob.NewDecoder(f)
	gob.Register(time.Time{})
	gob.Register(&cpu1.Cpu{})

	var pop census.Population
	if err := dec.Decode(&pop); err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}
	fmt.Printf("%+v\n", pop)
	fmt.Printf("%#v\n", pop)

	if c, ok := pop.Key.(*cpu1.Cpu); ok {
		slist, err := cpu1.Ops.Decompile(c.Code)
		if err != nil {
			fmt.Printf("error decompiling: %v\n", err)
		}
		for _, s := range slist {
			fmt.Println(s)
		}
	}
}
