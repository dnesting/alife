package term

import "github.com/dnesting/alife/goalife/energy"

type RuneFunc func(o interface{}) rune

var DefaultRunes RuneFunc = defaultFn

func defaultFn(o interface{}) rune {
	switch o := o.(type) {
	case *energy.Food:
		return RuneForFood(o, 5000)
	default:
		return '?'
	}
}

func RuneForFood(f *energy.Food, energyRange int) rune {
	e := f.Energy()
	switch {
	case e > energyRange/5*4:
		return '⁙'
	case e > energyRange/5*3:
		return '⁘'
	case e > energyRange/5*2:
		return '⁖'
	case e > energyRange/5*1:
		return '⁚'
	default:
		return '·'
	}
}
