package autosave

import "encoding/gob"
import "io/ioutil"
import "os"
import "path"
import "time"

import "github.com/dnesting/alife/goalife/grid2d"

func Save(filename string, g grid2d.Grid) error {
	dir := path.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := ioutil.TempFile(dir, "autosave")
	if err != nil {
		return err
	}
	defer f.Close()

	enc := gob.NewEncoder(f)
	if err := enc.Encode(g); err != nil {
		os.Remove(f.Name())
		return err
	}

	if err := os.Rename(f.Name(), filename); err != nil {
		os.Remove(f.Name())
		return err
	}

	return nil
}

func Restore(filename string, g grid2d.Grid) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	dec := gob.NewDecoder(f)
	if err := dec.Decode(g); err != nil {
		return err
	}
	return nil
}

func Loop(filename string, g grid2d.Grid, every time.Duration, exit <-chan bool) error {
	ch := time.Tick(every)
	for {
		select {
		case <-ch:
			if err := Save(filename, g); err != nil {
				return err
			}
		case <-exit:
			return nil
		}
	}
}
