package main

import "fmt"

import "entities"
import "entities/org"
import "entities/org/bank"
import "math/rand"
import "sim"
import "time"
import "world"

func putRandomOrg(w world.World) {
	o := org.Random()
	o.AddEnergy(3000)
	w.PutRandomlyIfEmpty(o)
}

const ensureOrgs = 100
const PrintEvery = 1

func main() {
	rand.Seed(time.Now().UnixNano())
	w := world.New(50, 100)
	w.ConsiderEmpty(func(o world.Occupant) bool {
		if _, ok := o.(*entities.Food); ok {
			return true
		}
		return false
	})
	w.PutRandomlyIfEmpty(entities.NewFood(1000))
	w.PutRandomlyIfEmpty(entities.NewFood(1000))
	w.PutRandomlyIfEmpty(entities.NewFood(1000))
	w.PutRandomlyIfEmpty(entities.NewFood(1000))
	putRandomOrg(w)

	s := &sim.Sim{w}

	bnk := bank.NewDirBank("/tmp/bank")

	var steps int
	fmt.Print("\033[H\033[2J")
	onUpdate := func(w world.World) error {
		steps++
		var count int
		survey := bank.NewSurvey()

		if steps%PrintEvery == 0 {
			fmt.Print("\033[H")
			fmt.Println(w)
			fmt.Printf("frame %d\n", steps)
		}

		w.Each(func(x, y int, o world.Occupant) {
			if org, ok := o.(*org.Organism); ok {
				count++
				survey.Record(org.Code)
			}
		})
		bnk.Record(steps, survey)

		if steps%PrintEvery == 0 {
			fmt.Printf("seen %d (%d species, %d recorded)\n", survey.Count(), survey.Distinct(), bnk.NumRecorded)
		}

		for count < ensureOrgs {
			putRandomOrg(w)
			count++
		}

		return nil
	}

	s.Run(onUpdate)
}
