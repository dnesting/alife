package org

import "fmt"
import "errors"
import "math"
import "math/rand"

import "entities"

type Cpu struct {
	Ip    int
	Code  []byte
	Table []*Op

	A int
	B int
	C int
	D int
}

func (c *Cpu) String() string {
	return fmt.Sprintf("[cpu ip=%d %d,%d,%d,%d]", c.Ip, c.A, c.B, c.C, c.D)
}

func NewCpu() *Cpu {
	return &Cpu{
		Table: OpTable,
	}
}

func NewCpuFrom(c *Cpu) *Cpu {
	return &Cpu{
		Table: c.Table,
	}
}

func (c *Cpu) SetCode(data []byte) (orig []byte) {
	orig = c.Code
	c.Code = data
	return
}

func OpNone(o *Organism, c *Cpu) error {
	return nil
}

func OpSwapAB(o *Organism, c *Cpu) error {
	c.A, c.B = c.B, c.A
	return nil
}

func OpSwapAC(o *Organism, c *Cpu) error {
	c.A, c.C = c.C, c.A
	return nil
}

func OpSwapAD(o *Organism, c *Cpu) error {
	c.A, c.D = c.D, c.A
	return nil
}

func OpZero(o *Organism, c *Cpu) error {
	c.A = 0
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

func OpShl(o *Organism, c *Cpu) error {
	c.A = asUByte(c.A << 1)
	return nil
}

func OpShr(o *Organism, c *Cpu) error {
	c.A = asUByte(c.A >> 1)
	return nil
}

func OpInc(o *Organism, c *Cpu) error {
	c.A = asUByte(c.A + 1)
	return nil
}

func OpDec(o *Organism, c *Cpu) error {
	c.A = asUByte(c.A - 1)
	return nil
}

func OpIfLoop(o *Organism, c *Cpu) error {
	if c.C > 0 {
		c.C -= 1
	} else {
		c.Ip += 1
	}
	return nil
}

func OpJump(o *Organism, c *Cpu) error {
	c.Ip = c.D
	return nil
}

func OpEat(o *Organism, c *Cpu) error {
	n, x, y := o.Neighbor()
	if n != nil {
		if e, ok := n.(entities.Energetic); ok {
			amt := e.Consume(o.w, x, y, 100) // c.A)
			o.AddEnergy(amt)
		}
	}
	return nil
}

func OpLeft(o *Organism, c *Cpu) error {
	o.Left()
	return nil
}

func OpRight(o *Organism, c *Cpu) error {
	o.Right()
	return nil
}

func OpForward(o *Organism, c *Cpu) error {
	o.Forward()
	return nil
}

func OpSense(o *Organism, c *Cpu) error {
	c.A = clip(o.Sense(false), 0, 255)
	return nil
}

func OpSenseOthers(o *Organism, c *Cpu) error {
	c.A = clip(o.Sense(true), 0, 255)
	return nil
}

func OpAdd(o *Organism, c *Cpu) error {
	c.A = asUByte(c.A + c.B)
	return nil
}

func OpSub(o *Organism, c *Cpu) error {
	c.A = asUByte(c.A - c.B)
	return nil
}

func OpMul(o *Organism, c *Cpu) error {
	c.A = asUByte(c.A * c.B)
	return nil
}

func OpDiv(o *Organism, c *Cpu) error {
	c.A = asUByte(c.A / c.B)
	return nil
}

func OpAnd(o *Organism, c *Cpu) error {
	c.A = asUByte(c.A & c.B)
	return nil
}

func OpOr(o *Organism, c *Cpu) error {
	c.A = asUByte(c.A | c.B)
	return nil
}

func OpXor(o *Organism, c *Cpu) error {
	c.A = asUByte(c.A ^ c.B)
	return nil
}

func OpMod(o *Organism, c *Cpu) error {
	c.A = asUByte(c.A % c.B)
	return nil
}

func OpIfEq(o *Organism, c *Cpu) error {
	if !(c.A == c.B) {
		c.Ip += 1
	}
	return nil
}

func OpIfNe(o *Organism, c *Cpu) error {
	if !(c.A != c.B) {
		c.Ip += 1
	}
	return nil
}

func OpIfLt(o *Organism, c *Cpu) error {
	if !(c.A < c.B) {
		c.Ip += 1
	}
	return nil
}

func OpIfGt(o *Organism, c *Cpu) error {
	if !(c.A > c.B) {
		c.Ip += 1
	}
	return nil
}

func OpIfZ(o *Organism, c *Cpu) error {
	if !(c.A == 0) {
		c.Ip += 1
	}
	return nil
}

func OpIfNZ(o *Organism, c *Cpu) error {
	if !(c.A != 0) {
		c.Ip += 1
	}
	return nil
}

const MutationFlipProb = 0.01
const MutationInsProb = 0.01
const MutationDelProb = 0.01

func (c *Cpu) Mutated() []byte {
	d := make([]byte, len(c.Code))
	copy(d, c.Code)
	maxOp := len(c.Table)

	i := rand.Intn(len(d))
	l := int(math.Ceil(math.Abs(rand.NormFloat64() * 10)))
	if rand.Float32() < MutationFlipProb {
		d[i] = byte(rand.Intn(maxOp))
	}
	if rand.Float32() < MutationInsProb {
		n := make([]byte, len(d)+l)
		if i > 0 {
			copy(n[:i], d[:i])
		}
		ld := len(d)
		for j := i; j < i+l; j++ {
			n[j] = d[j%ld]
		}
		for j := i; j < ld; j++ {
			n[j+l] = d[j]
		}
		d = n
	}
	if rand.Float32() < MutationDelProb {
		n := make([]byte, len(d)-l)
		if i > 0 {
			copy(n[:i], d[:i])
		}
		ld := len(d)
		if i < ld-1 {
			copy(n[i:], d[i+l:])
		}
		d = n
	}
	return d
}

func OpDivide(o *Organism, c *Cpu) error {
	o.Divide(float32(c.A) / 256.0)
	return nil
}

func OpNoop(o *Organism, c *Cpu) error {
	return nil
}

type Op struct {
	Name string
	Fn   func(o *Organism, c *Cpu) error
	Cost int
}

var OpTable = []*Op{
	// 0
	&Op{"Noop", OpNoop, 1},
	&Op{"SwapAB", OpSwapAB, 1},
	&Op{"SwapAC", OpSwapAC, 1},
	&Op{"SwapAD", OpSwapAD, 1},

	&Op{"Zero", OpZero, 1},
	&Op{"Shl", OpShl, 1},
	&Op{"Shr", OpShr, 1},

	&Op{"Inc", OpInc, 1},
	// 8
	&Op{"Dec", OpDec, 1},

	&Op{"Add", OpAdd, 1},
	&Op{"Sub", OpSub, 1},
	&Op{"Div", OpDiv, 1},
	&Op{"Mul", OpMul, 1},

	&Op{"And", OpAnd, 1},
	&Op{"Or", OpOr, 1},
	&Op{"Xor", OpXor, 1},
	// 16
	&Op{"Mod", OpMod, 1},

	&Op{"IfEq", OpIfEq, 1},
	&Op{"IfNe", OpIfNe, 1},
	&Op{"IfGt", OpIfGt, 1},
	&Op{"IfLt", OpIfLt, 1},

	&Op{"IfZ", OpIfZ, 1},
	&Op{"IfNZ", OpIfNZ, 1},
	&Op{"IfLoop", OpIfLoop, 1},
	// 24
	&Op{"Jump", OpJump, 1},

	&Op{"Eat", OpEat, 10},
	&Op{"Left", OpLeft, 5},
	&Op{"Right", OpRight, 5},
	&Op{"Forward", OpForward, 50},

	&Op{"Divide", OpDivide, 10},
	&Op{"Sense", OpSense, 1},
	&Op{"SenseOthers", OpSenseOthers, 1},

	// 32
}

func (c *Cpu) Step(o *Organism) (err error) {
	op, ip := c.readOp()
	c.Ip = ip
	if op == nil {
		return errors.New("unable to read next instruction")
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	if err := op.Fn(o, c); err != nil {
		return err
	}

	if _, e := o.AddEnergy(-op.Cost); e == 0 {
		return errors.New("ran out of energy")
	}
	return nil
}

func (c *Cpu) readOp() (*Op, int) {
	if c.Ip < 0 || c.Ip >= len(c.Code) {
		return nil, c.Ip + 1
	}
	b := c.Code[c.Ip]
	if b < 0 || b > byte(len(c.Table)) {
		return nil, c.Ip + 1
	}
	return c.Table[b], c.Ip + 1
}

const RandLengthMax = 1000
const RandLengthMin = 50

func RandomBytecode() []byte {
	s := rand.Intn(RandLengthMax-RandLengthMin) + RandLengthMin
	d := make([]byte, s)
	maxOp := len(OpTable)
	for i := 0; i < s; i++ {
		d[i] = byte(rand.Intn(maxOp))
	}
	return d
}
