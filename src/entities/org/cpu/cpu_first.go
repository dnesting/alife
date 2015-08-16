package cpu

import "org"

type Cpu interface {
	Step(Organism o) error
}

type cpu struct {
	Ip   int
	Code []byte

	A int
	B int
	C int
	D int
}

func OpNone(o org.Organism, c cpu) error {
	return nil
}

func OpSwapAB(o org.Organism, c cpu) error {
	c.A, c.B = c.B, c.A
	return nil
}

func OpSwapAC(o org.Organism, c cpu) error {
	c.A, c.C = c.C, c.A
	return nil
}

func OpSwapAD(o org.Organism, c cpu) error {
	c.A, c.D = c.D, c.A
	return nil
}

func OpZero(o org.Organism, c cpu) error {
	c.A = 0
	return nil
}

func normalize(v, limit int) int {
	v = v % limit
	if v < 0 {
		v += limit
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

func OpShl(o org.Organism, c cpu) error {
	c.A = normalize(c.A<<1, 256)
	return nil
}

func OpShr(o org.Organism, c cpu) error {
	c.A = normalize(c.A>>1, 256)
	return nil
}

func OpInc(o org.Organism, c cpu) error {
	c.A = normalize(c.A + 1)
	return nil
}

func OpDec(o org.Organism, c cpu) error {
	c.A = normalize(c.A - 1)
	return nil
}

func OpIfLoop(o org.Organism, c cpu) error {
	if c.C > 0 {
		c.C -= 1
	} else {
		c.Ip += 1
	}
	return nil
}

func OpJump(o org.Organism, c cpu) error {
	c.Ip = c.D
}

func OpEat(o org.Organism, c cpu) error {
	x := o.Neighbor()
	if x != nil {
		if e, ok := x.(entities.Energetic); ok {
			amt := e.AddEnergy(-c.A)
			o.AddEnergy(amt)
		}
	}
	return nil
}

func OpLeft(o org.Organism, c cpu) error {
	o.Left()
	return nil
}

func OpRight(o org.Organism, c cpu) error {
	o.Right()
	return nil
}

func OpForward(o org.Organism, c cpu) error {
	o.Forward()
	return nil
}

func OpSense(o org.Organism, c cpu) error {
	c.A = 0
	x := o.Neighbor()
	if x != nil {
		if e, ok := x.(entities.Energetic); ok {
			amt := e.Energy()
			c.A = clip(amt, 0, 255)
		}
	}
}

func OpAdd(o org.Organism, c cpu) error {
	c.A = normalize(c.A + c.B)
	return nil
}

func OpSub(o org.Organism, c cpu) error {
	c.A = normalize(c.A - c.B)
	return nil
}

func OpMul(o org.Organism, c cpu) error {
	c.A = normalize(c.A * c.B)
	return nil
}

func OpDiv(o org.Organism, c cpu) error {
	c.A = normalize(c.A / c.B)
	return nil
}

func OpAnd(o org.Organism, c cpu) error {
	c.A = normalize(c.A & c.B)
	return nil
}

func OpOr(o org.Organism, c cpu) error {
	c.A = normalize(c.A | c.B)
	return nil
}

func OpXor(o org.Organism, c cpu) error {
	c.A = normalize(c.A ^ c.B)
	return nil
}

func OpMod(o org.Organism, c cpu) error {
	c.A = normalize(c.A % c.B)
	return nil
}

func OpIfEq(o org.Organism, c cpu) error {
	if !(c.A == c.B) {
		c.Ip += 1
	}
	return nil
}

func OpIfNe(o org.Organism, c cpu) error {
	if !(c.A != c.B) {
		c.Ip += 1
	}
	return nil
}

func OpIfLt(o org.Organism, c cpu) error {
	if !(c.A < c.B) {
		c.Ip += 1
	}
	return nil
}

func OpIfGt(o org.Organism, c cpu) error {
	if !(c.A > c.B) {
		c.Ip += 1
	}
	return nil
}

func OpIfZ(o org.Organism, c cpu) error {
	if !(c.A == 0) {
		c.Ip += 1
	}
	return nil
}

func OpIfNZ(o org.Organism, c cpu) error {
	if !(c.A != 0) {
		c.Ip += 1
	}
	return nil
}

type Op struct {
	Name string
	Fn   func(o org.Organism, c cpu) error
}

var OpTable = []Op{
	// 0
	Op{"Noop", Noop},
	Op{"SwapAB", OpSwapAB},
	Op{"SwapAC", OpSwapAC},
	Op{"SwapAD", OpSwapAD},

	Op{"Zero", OpZero},
	Op{"Shl", OpShl},
	Op{"Shr", OpShr},

	Op{"Inc", OpInc},
	// 8
	Op{"Dec", OpDec},

	Op{"Add", OpAdd},
	Op{"Sub", OpSub},
	Op{"Div", OpDiv},
	Op{"Mul", OpMul},

	Op{"And", OpAnd},
	Op{"Or", OpOr},
	Op{"Xor", OpXor},
	// 16
	Op{"Mod", OpMod},

	Op{"IfEq", OpIfEq},
	Op{"IfNe", OpIfNe},
	Op{"IfGt", OpIfGt},
	Op{"IfLt", OpIfLt},

	Op{"IfZ", OpIfZ},
	Op{"IfNZ", OpIfNZ},
	Op{"IfLoop", OpIfLoop},
	// 24
	Op{"Jump", OpJump},

	Op{"Eat", OpEat},
	Op{"Left", OpLeft},
	Op{"Right", OpRight},
	Op{"Forward", OpForward},

	Op{"Sense", OpSense},

	// 32
}

func (c *cpu) Step(Organism o) error {
	if op, ip := c.readOp(); op == nil {
		org.Die("unable to read next instruction")
	}
	return op.Fn(o, c)
}

func (c *cpu) readOp() (Op, int) {
	if c.Ip < 0 || c.Ip >= len(c.Code) {
		return nil, c.Ip + 1
	}
	c = c.Code[c.Ip]
	if c < 0 || c > len(opTable) {
		return nil, c.Ip + 1
	}
	return opTable[c], c.Ip + 1
}
