package cpu1

import "hash/crc32"
import "math"
import "math/rand"

// Bytecode represents the instructions the Cpu should execute.
type Bytecode []byte

func (c Bytecode) Hash() uint64 {
	return uint64(crc32.ChecksumIEEE(c))
}

func (c Bytecode) Len() int {
	return len(c.Bytes())
}

func (c *Bytecode) Bytes() []byte {
	return []byte(*c)
}

// RandLengthMax is the maximum length of randomly-generated code.
var RandLengthMax = 1000

// RandLengthMin is the minimum length of randomly-generated code.
var RandLengthMin = 50

// RandomBytecode returns randomly-generated bytecode that is plausibly
// executable.
func RandomBytecode(ops OpTable) Bytecode {
	s := rand.Intn(RandLengthMax-RandLengthMin) + RandLengthMin
	d := make([]byte, s)
	maxOp := ops.Len()
	for i := 0; i < s; i++ {
		d[i] = byte(rand.Intn(maxOp))
	}
	return Bytecode(d)
}

// Mutate randomly mutates the code.  Three types of mutations are supported:
// 1. A single instruction change
// 2. Deletion of a segment
// 3. Duplication of a segment
func (c *Bytecode) Mutate(ops OpTable) {
	var d []byte
	maxOp := ops.Len()

	var i int
	i = rand.Intn(c.Len())
	l := int(math.Ceil(math.Abs(rand.NormFloat64() * 5)))
	prob := rand.Float32()
	if prob < 0.333 && c.Len() > 0 {
		// Change a single instruction at i
		d = make([]byte, c.Len())
		copy(d, c.Bytes())
		d[i] = byte(rand.Intn(maxOp))

	} else if prob < 0.666 {
		// Duplicate a segment starting at i of length l
		d = make([]byte, c.Len()+l)
		copy(d[:i], c.Bytes()[:i])
		for j := i; j < i+l; j++ {
			d[j] = c.Bytes()[j%c.Len()]
		}
		copy(d[i+l:], c.Bytes()[i:])

	} else if c.Len() > 0 {
		// Delete a segment starting at i of length l
		if i+l > c.Len() {
			l = c.Len() - i
		}
		d = make([]byte, c.Len()-l)
		copy(d[:i], c.Bytes()[:i])
		copy(d[i:], c.Bytes()[i+l:])
	}
	// Replace the CPU's code only if the mutated version is non-empty
	if len(d) > 0 {
		*c = Bytecode(d)
	}
}

// Find locates the given value in the CPU's code slice, searching forward and wrapping around.
func (c Bytecode) find(value int, start int) int {
	for i := start; i < c.Len(); i++ {
		if c[i] == byte(value) {
			return i
		}
	}
	for i := 0; i < start; i++ {
		if c[i] == byte(value) {
			return i
		}
	}
	return 0
}

// FindBackward locates the given value in the CPU's code slice, searching backward and wrapping around.
func (c Bytecode) findBackward(value int, start int) int {
	for i := c.Len() - 1; i > start; i-- {
		if c[i] == byte(value) {
			return i
		}
	}
	for i := 0; i < start; i++ {
		if c[i] == byte(value) {
			return i
		}
	}
	return 0
}
