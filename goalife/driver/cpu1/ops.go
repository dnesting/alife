package cpu1

import "errors"
import "math/rand"

import "github.com/dnesting/alife/goalife/org"

const MutationRate = 0.01

var ErrDivisionByZero = errors.New("division by zero")

var Ops OpTable

func init() {
	Ops = OpTable([]Op{
		// 0
		Op{"XXX", OpNoop, 1},
		Op{"L1", OpNoop, 1},
		Op{"L2", OpNoop, 1},
		Op{"L3", OpNoop, 1},
		Op{"L4", OpNoop, 1},

		Op{"Jump1", OpJump1, 1},
		Op{"Jump2", OpJump2, 1},
		Op{"Jump3", OpJump3, 1},
		Op{"Jump4", OpJump4, 1},

		Op{"JumpR1", OpJumpR1, 1},
		Op{"JumpR2", OpJumpR2, 1},
		Op{"JumpR3", OpJumpR3, 1},
		Op{"JumpR4", OpJumpR4, 1},

		Op{"SwapAB", OpSwapAB, 1},
		Op{"SwapAC", OpSwapAC, 1},
		Op{"SwapAD", OpSwapAD, 1},

		Op{"Zero", OpZero, 1},
		Op{"Shl0", OpShl0, 1},
		Op{"Shl1", OpShl1, 1},
		Op{"Shr", OpShr, 1},

		Op{"Inc", OpInc, 1},
		Op{"Dec", OpDec, 1},

		Op{"Add", OpAdd, 1},
		Op{"Sub", OpSub, 1},
		Op{"Div", OpDiv, 1},
		Op{"Mul", OpMul, 1},

		Op{"And", OpAnd, 1},
		Op{"Or", OpOr, 1},
		Op{"Xor", OpXor, 1},
		Op{"Mod", OpMod, 1},

		Op{"IfEq", OpIfEq, 1},
		Op{"IfNe", OpIfNe, 1},
		Op{"IfGt", OpIfGt, 1},
		Op{"IfLt", OpIfLt, 1},

		Op{"IfZ", OpIfZ, 1},
		Op{"IfNZ", OpIfNZ, 1},
		Op{"IfLoop", OpIfLoop, 1},
		Op{"Jump", OpJump, 1},

		Op{"Eat", OpEat, 5},
		Op{"Left", OpLeft, 5},
		Op{"Right", OpRight, 5},
		Op{"Forward", OpForward, 10},

		Op{"Divide", OpDivide, 1},
		Op{"Sense", OpSense, 1},
		Op{"SenseOthers", OpSenseOthers, 1},
	})
}

func OpNone(o *org.Organism, c *Cpu) error {
	return nil
}

func OpSwapAB(o *org.Organism, c *Cpu) error {
	c.R[0], c.R[1] = c.R[1], c.R[0]
	return nil
}

func OpSwapAC(o *org.Organism, c *Cpu) error {
	c.R[0], c.R[2] = c.R[2], c.R[0]
	return nil
}

func OpSwapAD(o *org.Organism, c *Cpu) error {
	c.R[0], c.R[3] = c.R[3], c.R[0]
	return nil
}

func OpZero(o *org.Organism, c *Cpu) error {
	c.R[0] = 0
	return nil
}

func asUByte(v int) int {
	v = v % 256
	if v < 0 {
		v += 256
	}
	return v
}

func clip(v, min, max int) int {
	if v > max {
		v = max
	}
	if v < min {
		v = min
	}
	return v
}

func OpShl0(o *org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0] << 1)
	return nil
}

func OpShl1(o *org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0]<<1) | 1
	return nil
}

func OpShr(o *org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0] >> 1)
	return nil
}

func OpInc(o *org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0] + 1)
	return nil
}

func OpDec(o *org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0] - 1)
	return nil
}

func OpIfLoop(o *org.Organism, c *Cpu) error {
	if c.R[2] > 0 {
		c.R[2] -= 1
	} else {
		c.Ip += 1
	}
	return nil
}

func OpJump(o *org.Organism, c *Cpu) error {
	c.Ip = c.R[3]
	return nil
}

func OpEat(o *org.Organism, c *Cpu) error {
	if _, err := o.Eat(c.R[0]); err != nil {
		return err
	}
	return nil
}

func OpLeft(o *org.Organism, c *Cpu) error {
	o.Left()
	return nil
}

func OpRight(o *org.Organism, c *Cpu) error {
	o.Right()
	return nil
}

func OpForward(o *org.Organism, c *Cpu) error {
	if err := o.Forward(); err != nil && err != org.ErrNotEmpty {
		return err
	}
	return nil
}

func OpSense(o *org.Organism, c *Cpu) error {
	c.R[0] = clip(int(o.Sense(nil)), 0, 255)
	return nil
}

func OpSenseOthers(o *org.Organism, c *Cpu) error {
	filter := func(n interface{}) float64 {
		if org, ok := n.(*org.Organism); ok {
			if nc, ok := org.Driver.(*Cpu); ok {
				if c.Hash() == nc.Hash() {
					return 0.0
				}
			}
		}
		return 1.0
	}
	c.R[0] = clip(int(o.Sense(filter)), 0, 255)
	return nil
}

func OpAdd(o *org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0] + c.R[1])
	return nil
}

func OpSub(o *org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0] - c.R[1])
	return nil
}

func OpMul(o *org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0] * c.R[1])
	return nil
}

func OpDiv(o *org.Organism, c *Cpu) error {
	if c.R[1] == 0 {
		return ErrDivisionByZero
	}
	c.R[0] = asUByte(c.R[0] / c.R[1])
	return nil
}

func OpAnd(o *org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0] & c.R[1])
	return nil
}

func OpOr(o *org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0] | c.R[1])
	return nil
}

func OpXor(o *org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0] ^ c.R[1])
	return nil
}

func OpMod(o *org.Organism, c *Cpu) error {
	if c.R[1] == 0 {
		return ErrDivisionByZero
	}
	c.R[0] = asUByte(c.R[0] % c.R[1])
	return nil
}

func OpIfEq(o *org.Organism, c *Cpu) error {
	if !(c.R[0] == c.R[1]) {
		c.Ip += 1
	}
	return nil
}

func OpIfNe(o *org.Organism, c *Cpu) error {
	if !(c.R[0] != c.R[1]) {
		c.Ip += 1
	}
	return nil
}

func OpIfLt(o *org.Organism, c *Cpu) error {
	if !(c.R[0] < c.R[1]) {
		c.Ip += 1
	}
	return nil
}

func OpIfGt(o *org.Organism, c *Cpu) error {
	if !(c.R[0] > c.R[1]) {
		c.Ip += 1
	}
	return nil
}

func OpIfZ(o *org.Organism, c *Cpu) error {
	if !(c.R[0] == 0) {
		c.Ip += 1
	}
	return nil
}

func OpIfNZ(o *org.Organism, c *Cpu) error {
	if !(c.R[0] != 0) {
		c.Ip += 1
	}
	return nil
}

func OpDivide(o *org.Organism, c *Cpu) error {
	lenc := len(c.Code)
	if err := o.Discharge(lenc); err != nil {
		return err
	}
	nc := c.Copy()
	if rand.Float32() < MutationRate {
		nc.Mutate()
	}
	n, err := o.Divide(nc, float64(c.R[0])/256.0)
	if err == org.ErrNotEmpty {
		return nil
	}
	if err != nil {
		return err
	}
	go nc.Run(n)
	return nil
}

func OpNoop(o *org.Organism, c *Cpu) error {
	return nil
}

func OpJump1(o *org.Organism, c *Cpu) error {
	c.Ip = c.Code.find(1, c.Ip)
	return nil
}
func OpJump2(o *org.Organism, c *Cpu) error {
	c.Ip = c.Code.find(2, c.Ip)
	return nil
}
func OpJump3(o *org.Organism, c *Cpu) error {
	c.Ip = c.Code.find(3, c.Ip)
	return nil
}
func OpJump4(o *org.Organism, c *Cpu) error {
	c.Ip = c.Code.find(4, c.Ip)
	return nil
}
func OpJumpR1(o *org.Organism, c *Cpu) error {
	c.Ip = c.Code.findBackward(1, c.Ip)
	return nil
}
func OpJumpR2(o *org.Organism, c *Cpu) error {
	c.Ip = c.Code.findBackward(2, c.Ip)
	return nil
}
func OpJumpR3(o *org.Organism, c *Cpu) error {
	c.Ip = c.Code.findBackward(3, c.Ip)
	return nil
}
func OpJumpR4(o *org.Organism, c *Cpu) error {
	c.Ip = c.Code.findBackward(4, c.Ip)
	return nil
}
