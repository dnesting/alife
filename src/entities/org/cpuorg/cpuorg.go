package cpuorg

import "fmt"
import "hash/crc32"
import "math/rand"

import "entities/census"
import "entities/org"
import "sim"

type CpuOrganism struct {
	org.BaseOrganism
	Cpu Cpu
}

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
	return fmt.Sprintf("[org (%d,%d) e=%d g=%d %v]", o.X, o.Y, o.Energy(), o.Cpu.Genome(), &o.Cpu)
}

func (o *CpuOrganism) Genome() census.Genome {
	return CpuOrgGenome{code: o.Cpu.Code}
}

var runes string = "abcdefhijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func (o *CpuOrganism) Rune() rune {
	return rune(runes[int(o.Cpu.Genome())%len(runes)])
}

func Random() *CpuOrganism {
	o := &CpuOrganism{}
	o.Cpu.SetCode(RandomBytecode())
	o.Dir = rand.Intn(8)
	return o
}

func FromCode(code []string) *CpuOrganism {
	o := &CpuOrganism{}
	o.Cpu.SetCode(Compile(code))
	o.Dir = rand.Intn(8)
	return o
}

func (o *CpuOrganism) Step(s *sim.Sim) error {
	if err := o.Cpu.Step(s, o); err != nil {
		return err
	}
	return nil
}

func (o *CpuOrganism) Mutate() {
	o.Cpu.Mutate()
}

func (o *CpuOrganism) Run(s *sim.Sim) {
	for !s.IsStopped() {
		if err := o.Step(s); err != nil {
			o.Die(s, o, err.Error())
			return
		}
	}
}
