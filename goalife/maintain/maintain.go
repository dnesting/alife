package maintain

import "io/ioutil"
import "log"

import "github.com/dnesting/alife/goalife/world/grid2d"

var Logger = log.New(ioutil.Discard, "", log.LstdFlags|log.Lshortfile)

func Maintain(ch <-chan []grid2d.Update, counterFn func(o interface{}) bool, fn func(), keep int) {
	//ready := make(chan bool)
	//go func() {
	//ready <- true
	// loop(ch, counterFn, fn, keep)
	go loop(ch, counterFn, fn, keep)
	//}()
	//<-ready
	Logger.Printf("seeding initial %d items\n", keep)
	for i := 0; i < keep; i++ {
		fn()
	}
}

func loop(ch <-chan []grid2d.Update, counterFn func(o interface{}) bool, fn func(), keep int) {
	var count int
	for updates := range ch {
		for _, u := range updates {
			if u.IsAdd() || u.IsReplace() {
				if counterFn(u.New.V) {
					count++
					Logger.Printf("%v add counted toward total of %d\n", u.New.V, count)
				} else {
					Logger.Printf("%v add not counted\n", u.New.V)
				}
			}
			if u.IsRemove() || u.IsReplace() {
				if counterFn(u.Old.V) {
					count--
					if count < keep {
						Logger.Printf("%v remove counted toward total of %d, adding one\n", u.Old.V, count)
						go fn()
					} else {
						Logger.Printf("%v remove counted toward total of %d\n", u.Old.V, count)
					}
				} else {
					Logger.Printf("%v remove not counted\n", u.Old.V)
				}
			}
		}
	}
}
