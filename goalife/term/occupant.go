package term

import "github.com/dnesting/alife/goalife/grid2d/food"
import "github.com/dnesting/alife/goalife/grid2d/org"
import "github.com/dnesting/alife/goalife/grid2d/org/driver/cpu1"

func RuneForOccupant(o interface{}) rune {
	switch o := o.(type) {
	case *food.Food:
		return RuneForFood(o, 5000)
	case *org.Organism:
		return RuneForOrganism(o)
	default:
		return '?'
	}
}

func RuneForFood(f *food.Food, energyRange int) rune {
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
