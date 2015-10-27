// Package cpu1 contains an implementation of an org.Organism driver that
// drives the organism using a simple virtual machine.
package cpu1

import "errors"
import "fmt"

import "github.com/dnesting/alife/goalife/grid2d"
import "github.com/dnesting/alife/goalife/grid2d/org"
import "github.com/dnesting/alife/goalife/log"

var Logger = log.Null()

// Cpu is a simple 8-bit CPU with 4 registers and associated bytecode.
type Cpu struct {
	Ip   int // Instruction Pointer, an index into Code for the next instruction
	Code Bytecode
	R    [4]int // Registers, described as A B C and D in the opcodes
}

func (c *Cpu) String() string {
	return fmt.Sprintf("[cpu %x ip=%d %v]", c.Code.Hash(), c.Ip, c.R)
}

// Copy returns a new Cpu with the same Code.  The Cpu's instruction pointer
// and registers are not copied.
func (c *Cpu) Copy() *Cpu {
	return &Cpu{
		Code: c.Code,
	}
}

// Mutate causes the Cpu's Code to be mutated.
func (c *Cpu) Mutate() {
	Logger.Printf("%v.Mutate()", c)
	c.Code.Mutate(Ops)
}

// Hash identifies the Cpu by its bytecode.  This is used to establish the
// "genome" of the organism's driver so that the census can track the population
// running the same bytecode.
func (c *Cpu) Hash() uint64 {
	return c.Code.Hash()
}

// Random generates a Cpu with random bytecode.  Its instruction pointer and
// registers are initialized to zeros.
func Random() *Cpu {
	return &Cpu{
		Code: RandomBytecode(Ops),
	}
}

var unableToReadErr = errors.New("unable to read next instruction")

// Step executes one CPU operation.  Any error returned either assessing the
// operation's energy cost or executing it will be returned by this method.
// Execution is expected to cease (and the organism's Die method
// invoked) if an error is returned.
func (c *Cpu) Step(o *org.Organism) (err error) {
	op, ip := c.readOp()
	c.Ip = ip
	if op == nil {
		return unableToReadErr
	}
	Logger.Printf("%v.Step(%v): %v\n", c, o, op)

	// All operations cost at least 1 energy, to avoid infinite loops.
	if err := o.Discharge(1 + op.Cost); err != nil {
		return err
	}

	if err := op.Fn(o, c); err != nil {
		return err
	}

	return nil
}

// Run executes Step repeatedly, until Step returns an error, at which point this
// method will invoke o.Die and return.
func (c *Cpu) Run(o *org.Organism) error {
	Logger.Printf("%v.Run(%v)\n", c, o)
	for {
		if err := c.Step(o); err != nil {
			Logger.Printf("%v.Run: %v\n", c, err)
			o.Die()
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
	if b < 0 || b > byte(len(Ops)) {
		return nil, c.Ip + 1
	}
	return &Ops[b], c.Ip + 1
}

// StartAll finds all organisms driven by Cpu instances, and spawns a goroutine
// to begin executing each Cpu instance found.
func StartAll(g grid2d.Grid) {
	var locs []grid2d.Point
	g.Locations(&locs)
	for _, p := range locs {
		if o, ok := p.V.(*org.Organism); ok {
			if c, ok := o.Driver.(*Cpu); ok {
				go c.Run(o)
			}
		}
	}
}
