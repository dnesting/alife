package term

import "github.com/dnesting/alife/goalife/grid2d/food"
import "github.com/dnesting/alife/goalife/grid2d/org"
import "github.com/dnesting/alife/goalife/grid2d/org/cpu1"

// RuneForOccupant produces a rune for the thing occupying a grid2d cell.
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

// RuneForFood produces a rune for f, scaling the number of dots based on
// the "upper limit" given by energyRange.
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

// RuneForOrganism produces a rune for g based on the bytecode hash of the organism.
func RuneForOrganism(g *org.Organism) rune {
	if g.Driver == nil {
		return '?'
	}
	switch o := g.Driver.(type) {
	case *cpu1.Cpu:
		return rune(codeRunes[o.Code.Hash()%uint64(len(codeRunes))])
	default:
		return '?'
	}
}
