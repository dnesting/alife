package cpuorg

import "bytes"
import "math/rand"

import "entities/org"
import "sim"

type Op struct {
	Name string
	Fn   func(s *sim.Sim, o org.Organism, c *Cpu) error
	Cost int
}

var OpTable []Op

func init() {
	OpTable = []Op{
		// 0
		Op{"XXX", OpNoop, 0},
		Op{"L1", OpNoop, 0},
		Op{"L2", OpNoop, 0},
		Op{"L3", OpNoop, 0},
		Op{"L4", OpNoop, 0},

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
	}
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

func Compile(prog []string) []byte {
	m := make(map[string]int)
	for i, op := range OpTable {
		m[op.Name] = i
	}
	var b bytes.Buffer
	for _, s := range prog {
		b.WriteByte(byte(m[s]))
	}
	return b.Bytes()
}

func Decompile(code []byte) []string {
	var s []string
	for _, b := range code {
		if int(b) > len(OpTable) {
			b = 0
		}
		s = append(s, OpTable[b].Name)
	}
	return s
}

func OpNone(s *sim.Sim, o org.Organism, c *Cpu) error {
	return nil
}

func OpSwapAB(s *sim.Sim, o org.Organism, c *Cpu) error {
	c.R[0], c.R[1] = c.R[1], c.R[0]
	return nil
}

func OpSwapAC(s *sim.Sim, o org.Organism, c *Cpu) error {
	c.R[0], c.R[2] = c.R[2], c.R[0]
	return nil
}

func OpSwapAD(s *sim.Sim, o org.Organism, c *Cpu) error {
	c.R[0], c.R[3] = c.R[3], c.R[0]
	return nil
}

func OpZero(s *sim.Sim, o org.Organism, c *Cpu) error {
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

func OpShl0(s *sim.Sim, o org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0] << 1)
	return nil
}

func OpShl1(s *sim.Sim, o org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0]<<1) | 1
	return nil
}

func OpShr(s *sim.Sim, o org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0] >> 1)
	return nil
}

func OpInc(s *sim.Sim, o org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0] + 1)
	return nil
}

func OpDec(s *sim.Sim, o org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0] - 1)
	return nil
}

func OpIfLoop(s *sim.Sim, o org.Organism, c *Cpu) error {
	if c.R[2] > 0 {
		c.R[2] -= 1
	} else {
		c.Ip += 1
	}
	return nil
}

func OpJump(s *sim.Sim, o org.Organism, c *Cpu) error {
	c.Ip = c.R[3]
	return nil
}

func OpEat(s *sim.Sim, o org.Organism, c *Cpu) error {
	o.EatNeighbor(s, c.R[0])
	return nil
}

func OpLeft(s *sim.Sim, o org.Organism, c *Cpu) error {
	o.Left()
	return nil
}

func OpRight(s *sim.Sim, o org.Organism, c *Cpu) error {
	o.Right()
	return nil
}

func OpForward(s *sim.Sim, o org.Organism, c *Cpu) error {
	o.Forward(s)
	return nil
}

func OpSense(s *sim.Sim, o org.Organism, c *Cpu) error {
	c.R[0] = clip(int(o.Sense(s, nil)), 0, 255)
	return nil
}

func OpSenseOthers(s *sim.Sim, o org.Organism, c *Cpu) error {
	filter := func(n interface{}) float64 {
		if nc, ok := n.(*CpuOrganism); ok {
			if nc.Cpu.Genome() == c.Genome() {
				return 0.0
			}
		}
		return 1.0
	}
	c.R[0] = clip(int(o.Sense(s, filter)), 0, 255)
	return nil
}

func OpAdd(s *sim.Sim, o org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0] + c.R[1])
	return nil
}

func OpSub(s *sim.Sim, o org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0] - c.R[1])
	return nil
}

func OpMul(s *sim.Sim, o org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0] * c.R[1])
	return nil
}

func OpDiv(s *sim.Sim, o org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0] / c.R[1])
	return nil
}

func OpAnd(s *sim.Sim, o org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0] & c.R[1])
	return nil
}

func OpOr(s *sim.Sim, o org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0] | c.R[1])
	return nil
}

func OpXor(s *sim.Sim, o org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0] ^ c.R[1])
	return nil
}

func OpMod(s *sim.Sim, o org.Organism, c *Cpu) error {
	c.R[0] = asUByte(c.R[0] % c.R[1])
	return nil
}

func OpIfEq(s *sim.Sim, o org.Organism, c *Cpu) error {
	if !(c.R[0] == c.R[1]) {
		c.Ip += 1
	}
	return nil
}

func OpIfNe(s *sim.Sim, o org.Organism, c *Cpu) error {
	if !(c.R[0] != c.R[1]) {
		c.Ip += 1
	}
	return nil
}

func OpIfLt(s *sim.Sim, o org.Organism, c *Cpu) error {
	if !(c.R[0] < c.R[1]) {
		c.Ip += 1
	}
	return nil
}

func OpIfGt(s *sim.Sim, o org.Organism, c *Cpu) error {
	if !(c.R[0] > c.R[1]) {
		c.Ip += 1
	}
	return nil
}

func OpIfZ(s *sim.Sim, o org.Organism, c *Cpu) error {
	if !(c.R[0] == 0) {
		c.Ip += 1
	}
	return nil
}

func OpIfNZ(s *sim.Sim, o org.Organism, c *Cpu) error {
	if !(c.R[0] != 0) {
		c.Ip += 1
	}
	return nil
}

func OpDivide(s *sim.Sim, o org.Organism, c *Cpu) error {
	lenc := len(c.Code)
	if err := c.cost(o, lenc); err != nil {
		return err
	}
	n := &CpuOrganism{}
	n.Cpu.Code = make([]byte, lenc)
	copy(n.Cpu.Code, c.Code)
	o.Divide(s, float32(c.R[0])/256.0, n, &n.BaseOrganism)
	return nil
}

func OpNoop(s *sim.Sim, o org.Organism, c *Cpu) error {
	return nil
}

func OpJump1(s *sim.Sim, o org.Organism, c *Cpu) error {
	c.Ip = c.find(1)
	return nil
}
func OpJump2(s *sim.Sim, o org.Organism, c *Cpu) error {
	c.Ip = c.find(2)
	return nil
}
func OpJump3(s *sim.Sim, o org.Organism, c *Cpu) error {
	c.Ip = c.find(3)
	return nil
}
func OpJump4(s *sim.Sim, o org.Organism, c *Cpu) error {
	c.Ip = c.find(4)
	return nil
}
func OpJumpR1(s *sim.Sim, o org.Organism, c *Cpu) error {
	c.Ip = c.findBackward(1)
	return nil
}
func OpJumpR2(s *sim.Sim, o org.Organism, c *Cpu) error {
	c.Ip = c.findBackward(2)
	return nil
}
func OpJumpR3(s *sim.Sim, o org.Organism, c *Cpu) error {
	c.Ip = c.findBackward(3)
	return nil
}
func OpJumpR4(s *sim.Sim, o org.Organism, c *Cpu) error {
	c.Ip = c.findBackward(4)
	return nil
}

func (c *Cpu) readOp() (*Op, int) {
	c.Ip %= len(c.Code)
	if c.Ip < 0 {
		return nil, c.Ip + 1
	}
	b := c.Code[c.Ip]
	if b < 0 || b > byte(len(OpTable)) {
		return nil, c.Ip + 1
	}
	return &OpTable[b], c.Ip + 1
}
