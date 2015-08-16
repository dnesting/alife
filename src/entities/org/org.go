package org

import "entities"
import "entities/org/cpu"

type Organism struct {
	Energetic

	Genome uint32

	data []byte
	cpu  cpu.Cpu
}

func New() *Organism {
	data = cpu.RandomBytecode()
	return &Organism{
		entities.Energy(0),
		Genome: crc32.ChecksumIEEE(data),
		data:   data,
		cpu:    cpu.New(data),
	}
}

func (o *Organism) Step() error {
	return o.cpu.Step(o)
}

func (o *Organism) Forward() {
}

func (o *Organism) Right() {
}

func (o *Organism) Left() {
}

func (o *Organism) Neighbor() {
}
