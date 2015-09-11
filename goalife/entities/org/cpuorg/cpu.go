package cpuorg

import "fmt"
import "errors"
import "hash/crc32"
import "math"
import "math/rand"

import "github.com/dnesting/alife/goalife/entities/org"
import "github.com/dnesting/alife/goalife/sim"

// Cpu is a simple 8-bit CPU with 4 registers.
type Cpu struct {
	Ip   int    // Instruction Pointer, an index into Code for the next instruction
	Code []byte // Bytecode instructions

	R [4]int // Registers, described as A B C and D in the opcodes

	genome uint32 // Cache of the checksum of Code
}

func (c *Cpu) String() string {
	return fmt.Sprintf("[cpu ip=%d %v]", c.Ip, c.R)
}

// Mutate randomly mutates the code attached to the CPU.  Three types of mutations are supported:
// 1. A single instruction change
// 2. Deletion of a segment
// 3. Duplication of a segment
func (c *Cpu) Mutate() {
	var d []byte
	maxOp := len(OpTable)

	var i int
	i = rand.Intn(len(c.Code))
	l := int(math.Ceil(math.Abs(rand.NormFloat64() * 5)))
	prob := rand.Float32()
	if prob < 0.333 && len(c.Code) > 0 {
		// Change a single instruction at i
		d = make([]byte, len(c.Code))
		copy(d, c.Code)
		d[i] = byte(rand.Intn(maxOp))

	} else if prob < 0.666 {
		// Duplicate a segment starting at i of length l
		d = make([]byte, len(c.Code)+l)
		copy(d[:i], c.Code[:i])
		for j := i; j < i+l; j++ {
			d[j] = c.Code[j%len(c.Code)]
		}
		copy(d[i+l:], c.Code[i:])

	} else if len(c.Code) > 0 {
		// Delete a segment starting at i of length l
		if i+l > len(c.Code) {
			l = len(c.Code) - i
		}
		d = make([]byte, len(c.Code)-l)
		copy(d[:i], c.Code[:i])
		copy(d[i:], c.Code[i+l:])
	}
	// Replace the CPU's code only if the mutated version is non-empty
	if len(d) > 0 {
		c.SetCode(d)
	}
}

// SetCode changes the Code slice used by this CPU.
func (c *Cpu) SetCode(d []byte) {
	c.Code = d
	c.genome = 0
}

// Genome returns the hash of the CPU's code, thus describing the "genome" of the organism.
func (c *Cpu) Genome() uint32 {
	if c.genome == 0 {
		c.genome = crc32.ChecksumIEEE(c.Code)
	}
	return c.genome
}

// Find locates the given value in the CPU's code slice, searching forward and wrapping around.
func (c *Cpu) find(v int) int {
	for i := c.Ip; i < len(c.Code); i++ {
		if c.Code[i] == byte(v) {
			return i
		}
	}
	for i := 0; i < c.Ip; i++ {
		if c.Code[i] == byte(v) {
			return i
		}
	}
	return 0
}

// Find locates the given value in the CPU's code slice, searching forward and wrapping around.
func (c *Cpu) findBackward(v int) int {
	for i := len(c.Code) - 1; i > c.Ip; i-- {
		if c.Code[i] == byte(v) {
			return i
		}
	}
	for i := 0; i < c.Ip; i++ {
		if c.Code[i] == byte(v) {
			return i
		}
	}
	return 0
}

// Step executes one CPU operation.  Any error or panic that occurs will result in an error
// being returned.  Execution is expected to cease if an error is returned.
func (c *Cpu) Step(s *sim.Sim, o org.Organism) (err error) {
	op, ip := c.readOp()
	c.Ip = ip
	if op == nil {
		err := errors.New("unable to read next instruction")
		s.T(o, "step @%d (op): %v", ip, err)
		return err
	}

	if err := c.cost(o, op.Cost); err != nil {
		s.T(o, "step %v (cost): %v", op, err)
		return err
	}

	if err := op.Fn(s, o, c); err != nil {
		s.T(o, "step %v (fn): %v", op, err)
		return err
	}

	s.T(o, "step %v", op)
	return nil
}

// cost is a helper that applies an energy cost to the organism for an operation.
// Returns an error if the organism's energy level hits zero.
func (c *Cpu) cost(o org.Organism, amt int) error {
	if _, e := o.AddEnergy(-amt); e == 0 {
		return errors.New("ran out of energy")
	}
	return nil
}
