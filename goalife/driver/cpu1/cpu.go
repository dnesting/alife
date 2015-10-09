package cpu1

import "fmt"
import "errors"

import "github.com/dnesting/alife/goalife/org"

// Cpu is a simple 8-bit CPU with 4 registers.
type Cpu struct {
	Ip   int // Instruction Pointer, an index into Code for the next instruction
	Code Bytecode
	R    [4]int // Registers, described as A B C and D in the opcodes
}

func (c *Cpu) String() string {
	return fmt.Sprintf("[cpu ip=%d %v]", c.Ip, c.R)
}

func (c *Cpu) Copy() *Cpu {
	return &Cpu{
		Code: c.Code.Copy(),
	}
}

func (c *Cpu) Mutate() {
	c.Code.Mutate(opTable)
}

func (c *Cpu) Hash() uint64 {
	return c.Code.Hash()
}

func Random() *Cpu {
	return &Cpu{
		Code: RandomBytecode(opTable),
	}
}

// Step executes one CPU operation.  Any error or panic that occurs will result in an error
// being returned.  Execution is expected to cease if an error is returned.
func (c *Cpu) Step(o *org.Organism) (err error) {
	op, ip := c.readOp()
	c.Ip = ip
	if op == nil {
		err := errors.New("unable to read next instruction")
		return err
	}

	if err := o.Discharge(op.Cost); err != nil {
		return err
	}

	if err := op.Fn(o, c); err != nil {
		return err
	}

	return nil
}

func (c *Cpu) Run(o *org.Organism) error {
	for {
		if err := c.Step(o); err != nil {
			return err
		}
	}
}

func (c *Cpu) readOp() (*Op, int) {
	c.Ip %= len(c.Code)
	if c.Ip < 0 {
		return nil, c.Ip + 1
	}
	b := c.Code[c.Ip]
	if b < 0 || b > byte(len(opTable)) {
		return nil, c.Ip + 1
	}
	return &opTable[b], c.Ip + 1
}
