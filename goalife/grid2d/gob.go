package grid2d

import "bytes"
import "encoding/gob"

type gobStruct struct {
	Width  int
	Height int
	Points []Point
}

var gobData gobStruct

func (g *grid) GobEncode() ([]byte, error) {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	width, height, _ := g.Locations(&gobData.Points)
	gobData.Width = width
	gobData.Height = height
	if err := enc.Encode(gobData); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (g *grid) GobDecode(data []byte) error {
	b := bytes.NewBuffer(data)
	dec := gob.NewDecoder(b)
	var gs gobStruct
	if err := dec.Decode(&gs); err != nil {
		return err
	}
	g.Resize(gs.Width, gs.Height, nil)
	for _, p := range gs.Points {
		g.Put(p.X, p.Y, p.V, PutAlways)
	}
	return nil
}
