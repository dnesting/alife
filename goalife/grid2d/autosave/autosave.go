// Package autosave provides a method for saving and storing a grid2d.
package autosave

import "encoding/gob"
import "io/ioutil"
import "os"
import "path"
import "time"

import "github.com/dnesting/alife/goalife/grid2d"

// Save writes g to filename.
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

// Restore restores the contents of g from filename.
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

// Loop calls Save every freq.  Stops saving when exit yields a value.
func Loop(filename string, g grid2d.Grid, freq time.Duration, exit <-chan bool) error {
	ch := time.Tick(freq)
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
