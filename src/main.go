package main

import "fmt"

import "entities"
import "sim"
import "world"

func main() {
	w := world.New(10, 10)
	w.Put(1, 1, entities.NewFood(3))
	w.Put(2, 2, entities.NewFood(3))
	w.Put(3, 3, entities.NewFood(2))
	w.Put(4, 4, entities.NewFood(2))

	s := &sim.Sim{w}

	var steps int
	onUpdate := func(w world.World) error {
		fmt.Println(w)
		steps++
		if steps > 100 {
			return sim.StopRunning
		}
		return nil
	}

	s.Run(onUpdate)
}
