// Package cpuorg contains an implementation of an Organism that executes operations
// using a virtual CPU.
package cpuorg

import "fmt"
import "hash/crc32"
import "math/rand"
import "runtime"

import "github.com/dnesting/alife/goalife/entities/census"
import "github.com/dnesting/alife/goalife/entities/org"
import "github.com/dnesting/alife/goalife/sim"

// CpuOrganism is an Organism that executes operations using a CPU.
type CpuOrganism struct {
	org.BaseOrganism
	Cpu Cpu
}

// CpuOrgGenome represents a census.Genome derived from the CpuOrgGenome's code.
type CpuOrgGenome struct {
	hash       uint32
	code       []byte
	decompiled []string
}

func (g CpuOrgGenome) Hash() uint32 {
	if g.hash == 0 {
		g.hash = crc32.ChecksumIEEE(g.code)
	}
	return g.hash
}

func (g CpuOrgGenome) Code() []string {
	if g.decompiled == nil {
		g.decompiled = Decompile(g.code)
	}
	return g.decompiled
}

func (o *CpuOrganism) String() string {
	return fmt.Sprintf("[org (%s) e=%d g=%d %v]", o.BaseOrganism.Loc, o.Energy(), o.Cpu.Genome(), &o.Cpu)
}

// Genome returns a census.Genome corresponding to this organism.
func (o *CpuOrganism) Genome() census.Genome {
	return CpuOrgGenome{code: o.Cpu.Code}
}

var runes string = "abcdefhijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// Rune contains the rendering of this genome for the terminal, using alphanumeric characters
// tied to the organism's genome.
func (o *CpuOrganism) Rune() rune {
	return rune(runes[int(o.Cpu.Genome())%len(runes)])
}

// Random creates a randomly-generated organism, with random instructions and direction.
func Random() *CpuOrganism {
	o := &CpuOrganism{}
	o.Cpu.SetCode(RandomBytecode())
	o.Dir = rand.Intn(8)
	return o
}

// FromCode creates a new organism with the provided symbolic code and a random direction.
func FromCode(code []string) *CpuOrganism {
	o := &CpuOrganism{}
	o.Cpu.SetCode(Compile(code))
	o.Dir = rand.Intn(8)
	return o
}

// Step executes a single CPU instruction. Any error occurring during execution of the
// instruction will be returned, at which point the organism is not expected to continue
// executing.
func (o *CpuOrganism) Step(s *sim.Sim) error {
	if err := o.Cpu.Step(s, o); err != nil {
		return err
	}
	return nil
}

// Mutate mutates the CPU code of this organism.
func (o *CpuOrganism) Mutate() {
	o.Cpu.Mutate()
}

// Run continuously executes CPU instructions until the simulation is stopped.
// If an error occurs executing an instruction, the organism is killed and execution
// halted.
func (o *CpuOrganism) Run(s *sim.Sim) {
	s.T(o, "run")
	defer func() { s.T(o, "run exiting") }()

	for !s.IsStopped() {
		if err := o.Step(s); err != nil {
			o.Die(s, o, err.Error())
			return
		}
		runtime.Gosched()
	}
}
