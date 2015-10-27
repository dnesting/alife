package cpu1

import "errors"
import "math/rand"

import "github.com/dnesting/alife/goalife/grid2d/org"

// MutationRate specifies the rate at which mutations occur during a Divide operation.
var MutationRate = 0.01

// ErrDivisionByZero is reported when an opcode would result in division by zero.
var ErrDivisionByZero = errors.New("division by zero")

// ops contains the actual optable for cpu1.
var Ops OpTable

func init() {
	// Note: Modifying opcodes risks making any organisms saved by the census nonviable.
	Ops = OpTable([]Op{
		Op{"XXX", opNoop, 0},

		// L1-L4 represent labels used by opJumpN and opJumpRN opcodes.
		Op{"L1", opNoop, 0},
		Op{"L2", opNoop, 0},
		Op{"L3", opNoop, 0},
		Op{"L4", opNoop, 0},

		Op{"Jump1", opJump1, 0},
		Op{"Jump2", opJump2, 0},
		Op{"Jump3", opJump3, 0},
		Op{"Jump4", opJump4, 0},

		Op{"JumpR1", opJumpR1, 0},
		Op{"JumpR2", opJumpR2, 0},
		Op{"JumpR3", opJumpR3, 0},
		Op{"JumpR4", opJumpR4, 0},

		Op{"SwapAB", opSwapAB, 0},
		Op{"SwapAC", opSwapAC, 0},
		Op{"SwapAD", opSwapAD, 0},

		Op{"Zero", opZero, 0},
		Op{"Shl0", opShl0, 0},
		Op{"Shl1", opShl1, 0},
		Op{"Shr", opShr, 0},

		Op{"Inc", opInc, 0},
		Op{"Dec", opDec, 0},

		Op{"Add", opAdd, 0},
		Op{"Sub", opSub, 0},
		Op{"Div", opDiv, 0},
		Op{"Mul", opMul, 0},

		Op{"And", opAnd, 0},
		Op{"Or", opOr, 0},
		Op{"Xor", opXor, 0},
		Op{"Mod", opMod, 0},

		Op{"IfEq", opIfEq, 0},
		Op{"IfNe", opIfNe, 0},
		Op{"IfGt", opIfGt, 0},
		Op{"IfLt", opIfLt, 0},

		Op{"IfZ", opIfZ, 0},
		Op{"IfNZ", opIfNZ, 0},
		Op{"IfLoop", opIfLoop, 0},
		Op{"Jump", opJump, 0},

		Op{"Eat", opEat, 5},
		Op{"Left", opLeft, 5},
		Op{"Right", opRight, 5},
		Op{"Forward", opForward, 10},

		Op{"Divide", opDivide, 0},
		Op{"Sense", opSense, 0},
		Op{"SenseOthers", opSenseOthers, 0},
	})
}

// opSwapAB: A, B = B, A
func opSwapAB(o *org.Organism, c *Cpu) error {
	c.R[0], c.R[1] = c.R[1], c.R[0]
	return nil
}

// opSwapAC: A, C = C, A
func opSwapAC(o *org.Organism, c *Cpu) error {
	c.R[0], c.R[2] = c.R[2], c.R[0]
	return nil
}

// opSwapAD: A, D = D, A
func opSwapAD(o *org.Organism, c *Cpu) error {
	c.R[0], c.R[3] = c.R[3], c.R[0]
	return nil
}

// opZero: A = 0
func opZero(o *org.Organism, c *Cpu) error {
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

// opShl0: A <<= 1
func opShl0(o *org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0] << 1)
	return nil
}

// opShl1: A = A<<1 | 1
func opShl1(o *org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0]<<1) | 1
	return nil
}

// opShr: A >>= 1
func opShr(o *org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0] >> 1)
	return nil
}

// opInc: A++
func opInc(o *org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0] + 1)
	return nil
}

// opDec: A--
func opDec(o *org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0] - 1)
	return nil
}

// opIfLoop: if C > 0 { execute next instruction } else skip
func opIfLoop(o *org.Organism, c *Cpu) error {
	if c.R[2] > 0 {
		c.R[2] -= 1
	} else {
		c.Ip += 1
	}
	return nil
}

// opJump: IP = D
func opJump(o *org.Organism, c *Cpu) error {
	c.Ip = c.R[3]
	return nil
}

// opEat: Consume A*10 energy from neighbor
func opEat(o *org.Organism, c *Cpu) error {
	if _, err := o.Eat(c.R[0] * 10); err != nil {
		return err
	}
	return nil
}

// opLeft: turn left
func opLeft(o *org.Organism, c *Cpu) error {
	o.Left()
	return nil
}

// opRight: turn right
func opRight(o *org.Organism, c *Cpu) error {
	o.Right()
	return nil
}

// opForward: move forward if able
func opForward(o *org.Organism, c *Cpu) error {
	if err := o.Forward(); err != nil && err != org.ErrNotEmpty {
		return err
	}
	return nil
}

// opSense: sense energy ahead, capped at 255
func opSense(o *org.Organism, c *Cpu) error {
	c.R[0] = clip(int(o.Sense(nil)), 0, 255)
	return nil
}

// opSenseOthers: sense energy ahead, excluding those with the same bytecode, capped at 255
func opSenseOthers(o *org.Organism, c *Cpu) error {
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

// opAdd: A += B
func opAdd(o *org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0] + c.R[1])
	return nil
}

// opSub: A -= B
func opSub(o *org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0] - c.R[1])
	return nil
}

// opMul: A *= B
func opMul(o *org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0] * c.R[1])
	return nil
}

// opDiv: A /= B (may return ErrDivisionByZero)
func opDiv(o *org.Organism, c *Cpu) error {
	if c.R[1] == 0 {
		return ErrDivisionByZero
	}
	c.R[0] = asUByte(c.R[0] / c.R[1])
	return nil
}

// opAnd: A &= B
func opAnd(o *org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0] & c.R[1])
	return nil
}

// opOr: A |= B
func opOr(o *org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0] | c.R[1])
	return nil
}

// opXor: A ^= B
func opXor(o *org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0] ^ c.R[1])
	return nil
}

// opMod: A %= B (may return ErrDivisionByZero)
func opMod(o *org.Organism, c *Cpu) error {
	if c.R[1] == 0 {
		return ErrDivisionByZero
	}
	c.R[0] = asUByte(c.R[0] % c.R[1])
	return nil
}

// opIfEq: if A == B { execute next instruction } else skip
func opIfEq(o *org.Organism, c *Cpu) error {
	if !(c.R[0] == c.R[1]) {
		c.Ip += 1
	}
	return nil
}

// opIfNe: if A != B { execute next instruction } else skip
func opIfNe(o *org.Organism, c *Cpu) error {
	if !(c.R[0] != c.R[1]) {
		c.Ip += 1
	}
	return nil
}

// opIfLt: if A < B { execute next instruction } else skip
func opIfLt(o *org.Organism, c *Cpu) error {
	if !(c.R[0] < c.R[1]) {
		c.Ip += 1
	}
	return nil
}

// opIfGt: if A > B { execute next instruction } else skip
func opIfGt(o *org.Organism, c *Cpu) error {
	if !(c.R[0] > c.R[1]) {
		c.Ip += 1
	}
	return nil
}

// opIfZ: if A == 0 { execute next instruction } else skip
func opIfZ(o *org.Organism, c *Cpu) error {
	if !(c.R[0] == 0) {
		c.Ip += 1
	}
	return nil
}

// opIfNZ: if A != 0 { execute next instruction } else skip
func opIfNZ(o *org.Organism, c *Cpu) error {
	if !(c.R[0] != 0) {
		c.Ip += 1
	}
	return nil
}

// opDivide: spawn a new organism in neighboring cell with same bytecode
// and energy fraction described by A/256.
func opDivide(o *org.Organism, c *Cpu) error {
	lenc := len(c.Code)
	if err := o.Discharge(lenc); err != nil {
		return err
	}
	nc := c.Copy()
	if rand.Float64() < MutationRate {
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

// opNoop: No-op
func opNoop(o *org.Organism, c *Cpu) error {
	return nil
}

// opJump1: Jump forward to label A
func opJump1(o *org.Organism, c *Cpu) error {
	c.Ip = c.Code.find(1, c.Ip)
	return nil
}

// opJump2: Jump forward to label B
func opJump2(o *org.Organism, c *Cpu) error {
	c.Ip = c.Code.find(2, c.Ip)
	return nil
}

// opJump3: Jump forward to label C
func opJump3(o *org.Organism, c *Cpu) error {
	c.Ip = c.Code.find(3, c.Ip)
	return nil
}

// opJump4: Jump forward to label D
func opJump4(o *org.Organism, c *Cpu) error {
	c.Ip = c.Code.find(4, c.Ip)
	return nil
}

// opJumpR1: Jump backward to label A
func opJumpR1(o *org.Organism, c *Cpu) error {
	c.Ip = c.Code.findBackward(1, c.Ip)
	return nil
}

// opJumpR2: Jump backward to label B
func opJumpR2(o *org.Organism, c *Cpu) error {
	c.Ip = c.Code.findBackward(2, c.Ip)
	return nil
}

// opJumpR3: Jump backward to label C
func opJumpR3(o *org.Organism, c *Cpu) error {
	c.Ip = c.Code.findBackward(3, c.Ip)
	return nil
}

// opJumpR4: Jump backward to label D
func opJumpR4(o *org.Organism, c *Cpu) error {
	c.Ip = c.Code.findBackward(4, c.Ip)
	return nil
}
