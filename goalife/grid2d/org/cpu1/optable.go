package cpu1

import "bytes"
import "fmt"

import "github.com/dnesting/alife/goalife/grid2d/org"

// op represents a single named instruction.
type Op struct {
	Name string                              // the symbolic name of the instruction
	Fn   func(o *org.Organism, c *Cpu) error // what gets executed for this instruction
	Cost int                                 // the instruction's energy cost (above the default 1)
}

func (o Op) String() string {
	return o.Name
}

type OpTable []Op

func (ops OpTable) Len() int {
	return len([]Op(ops))
}

type UnknownOpErr struct {
	V interface{}
}

func (e UnknownOpErr) Error() string {
	return fmt.Sprintf("unknown operation: %v", e.V)
}

// Compile converts a slice of symbolic instructions into bytecode.
func (ops OpTable) Compile(prog []string) (Bytecode, error) {
	m := make(map[string]byte)
	for i, op := range ops {
		m[op.Name] = byte(i)
	}
	var buf bytes.Buffer
	for _, s := range prog {
		if b, ok := m[s]; ok {
			buf.WriteByte(b)
		} else {
			return nil, UnknownOpErr{s}
		}
	}
	return Bytecode(buf.Bytes()), nil
}

// Decompile converts a slice of bytecode into symbolic instructions.
func (ops OpTable) Decompile(code []byte) ([]string, error) {
	var s []string
	for _, b := range code {
		if int(b) < ops.Len() {
			s = append(s, ops[b].Name)
		} else {
			return nil, UnknownOpErr{b}
		}
	}
	return s, nil
}
