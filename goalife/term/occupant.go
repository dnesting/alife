package term

import "github.com/dnesting/alife/goalife/energy"
import "github.com/dnesting/alife/goalife/org"
import "github.com/dnesting/alife/goalife/driver/cpu1"

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

var codeRunes string = "abcdefhijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func RuneForOrganism(f *org.Organism) rune {
	if f.Driver == nil {
		return '?'
	}
	switch o := f.Driver.(type) {
	case *cpu1.Cpu:
		return rune(codeRunes[o.Code.Hash()%uint64(len(codeRunes))])
	default:
		return '?'
	}
}
