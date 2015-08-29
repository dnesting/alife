package cpuorg

import "strings"
import "runtime/debug"
import "fmt"
import "errors"
import "hash/crc32"
import "math"
import "math/rand"

import "entities/org"
import "sim"

type Cpu struct {
	Ip   int
	Code []byte

	R [4]int

	genome uint32
}

func (c *Cpu) String() string {
	return fmt.Sprintf("[cpu ip=%d %v]", c.Ip, c.R)
}

func (c *Cpu) Mutate() {
	var d []byte
	maxOp := len(OpTable)

	var i int
	i = rand.Intn(len(c.Code))
	l := int(math.Ceil(math.Abs(rand.NormFloat64() * 5)))
	prob := rand.Float32()
	if prob < 0.333 && len(c.Code) > 0 {
		d = make([]byte, len(c.Code))
		copy(d, c.Code)
		d[i] = byte(rand.Intn(maxOp))

	} else if prob < 0.666 {
		d = make([]byte, len(c.Code)+l)
		copy(d[:i], c.Code[:i])
		for j := i; j < i+l; j++ {
			d[j] = c.Code[j%len(c.Code)]
		}
		copy(d[i+l:], c.Code[i:])

	} else if len(c.Code) > 0 {
		if i+l > len(c.Code) {
			l = len(c.Code) - i
		}
		d = make([]byte, len(c.Code)-l)
		copy(d[:i], c.Code[:i])
		copy(d[i:], c.Code[i+l:])
	}
	if len(d) > 0 {
		c.SetCode(d)
	}
}

func (c *Cpu) SetCode(d []byte) {
	c.Code = d
	c.genome = 0
}

func (c *Cpu) Genome() uint32 {
	if c.genome == 0 {
		c.genome = crc32.ChecksumIEEE(c.Code)
	}
	return c.genome
}

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

func (c *Cpu) Step(s *sim.Sim, o org.Organism) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
			if strings.Contains(err.Error(), "signal") {
				debug.PrintStack()
			}
		}
	}()

	op, ip := c.readOp()
	c.Ip = ip
	if op == nil {
		return errors.New("unable to read next instruction")
	}
	//fmt.Printf("%s %s\n", c, op.Name)

	if err := op.Fn(s, o, c); err != nil {
		return err
	}

	if err := c.cost(o, op.Cost); err != nil {
		return err
	}
	return nil
}

func (c *Cpu) cost(o org.Organism, amt int) error {
	if _, e := o.AddEnergy(-amt); e == 0 {
		return errors.New("ran out of energy")
	}
	return nil
}
