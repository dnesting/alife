package census

import "bytes"
import "encoding/gob"
import "io"
import "os"
import "path"
import "testing"
import "time"

type closeBuffer struct {
	bytes.Buffer
	Closed bool
}

func (c *closeBuffer) Close() error {
	c.Closed = true
	return nil
}

func encoded(t *testing.T, p Population) io.ReadWriteCloser {
	fk := fakeKey(0)
	gob.Register(fk)
	var b closeBuffer
	enc := gob.NewEncoder(&b.Buffer)
	if err := enc.Encode(p); err != nil {
		t.Fatalf("unable to encode %v: %v", p, err)
	}
	return &b
}

func decoded(t *testing.T, b *closeBuffer) Population {
	var p Population
	dec := gob.NewDecoder(b)
	if err := dec.Decode(&p); err != nil {
		t.Fatalf("unable to decode %v: %v", p, err)
	}
	return p
}

type fi struct {
	N string
}

func (f fi) Name() string       { return f.N }
func (f fi) Size() int64        { return 0 }
func (f fi) Mode() os.FileMode  { return os.FileMode(0777) }
func (f fi) ModTime() time.Time { return time.Now() }
func (f fi) IsDir() bool        { return false }
func (f fi) Sys() interface{}   { return nil }

func TestNew(t *testing.T) {
	dir := "/path/foo"
	deps.ReadDir = func(s string) ([]os.FileInfo, error) {
		if s != dir {
			t.Errorf("ReadDir called with wrong path, expected %s, got %s", dir, s)
		}
		return []os.FileInfo{fi{"a"}, fi{"b"}, fi{"c"}}, nil
	}
	deps.MkdirAll = func(_ string, _ os.FileMode) error { return nil }

	d, err := NewDirCensus(dir, nil)
	if err != nil {
		t.Errorf("unexpected error creating dir census: %v", err)
	}
	if d.NumRecorded() != 3 {
		t.Errorf("NumRecorded() expected %d, got %d", 3, d.NumRecorded())
	}
}

func TestGetFromRecord(t *testing.T) {
	dir := "/path/foo"
	key := fakeKey(0x100)
	file := path.Join(dir, "100")
	badKey := fakeKey(0x101)
	badFile := path.Join(dir, "101")

	f := encoded(t, Population{
		Key:   key,
		Count: 10,
	})

	deps.Open = func(s string) (io.ReadWriteCloser, error) {
		if s == file {
			return f, nil
		} else {
			if s != badFile {
				t.Errorf("Open called with unexpected filename, expected /path/foo/{100,101}, got %v", s)
			}
			return nil, os.ErrNotExist
		}
	}

	c := DirCensus{
		Dir: dir,
	}
	if _, err := c.GetFromRecord(badKey); err != os.ErrNotExist {
		t.Errorf("GetFromRecord with bad key should generate ErrNotFound, got %v", err)
	}
	p, err := c.GetFromRecord(key)
	if err != nil {
		t.Errorf("got error %v reading valid key", err)
	}
	if p.Count != 10 {
		t.Errorf("retrieved count was wrong, expected 10, got %v", p.Count)
	}
}

func TestIsRecorded(t *testing.T) {
	dir := "/path/foo"
	key := fakeKey(0x100)
	file := path.Join(dir, "100")
	badKey := fakeKey(0x101)
	badFile := path.Join(dir, "101")

	deps.Stat = func(s string) (os.FileInfo, error) {
		if s == file {
			return fi{file}, nil
		} else if s == badFile {
			return fi{}, os.ErrNotExist
		}
		t.Errorf("Stat called with unexpected file, expected /path/foo/{100,101}, got %v", s)
		return fi{}, os.ErrNotExist
	}

	c := DirCensus{Dir: dir}
	if !c.IsRecorded(key) {
		t.Errorf("IsRecorded(%v) should be true but was not", key)
	}
	if c.IsRecorded(badKey) {
		t.Errorf("IsRecorded(%v) should be false but was not", badKey)
	}
}

func TestRecord(t *testing.T) {
	dir := "/path/foo"
	key := fakeKey(0x100)
	key.Other = 42
	file := path.Join(dir, "100")
	pop := Population{Key: key, Count: 10}
	b := &closeBuffer{}

	deps.Create = func(s string) (io.ReadWriteCloser, error) {
		if s == file {
			return b, nil
		}
		t.Errorf("Create called with unexpected filename, wanted %v got %v", file, s)
		return nil, os.ErrNotExist
	}

	c := DirCensus{Dir: dir}
	err := c.Record(pop)
	if err != nil {
		t.Errorf("Record should not have resulted in error, got %v", err)
	}
	p := decoded(t, b)
	if p.Count != 10 {
		t.Errorf("Count should be 10, got %v", p.Count)
	}
	if fk, ok := p.Key.(fakeKeyType); ok {
		if fk.Other != 42 {
			t.Errorf("key fields did not survive encoding, expected other=42, got %+v", fk)
		}
	} else {
		t.Errorf("key type did not survive encoding, expected fakeKeyType, got %+v", p.Key)
	}
}

func TestAdd(t *testing.T) {
	dir := "/path/foo"
	key1 := fakeKey(0x100)
	key2 := fakeKey(0x101)
	file2 := path.Join(dir, "101")
	filt := func(p Population) bool { return p.Count > 2 }

	var ok bool
	b := &closeBuffer{}
	deps.Stat = func(s string) (os.FileInfo, error) { return nil, os.ErrNotExist }
	deps.Create = func(s string) (io.ReadWriteCloser, error) {
		if s == file2 {
			ok = true
			return b, nil
		}
		t.Errorf("Create called with unexpected filename, wanted %v got %v", file2, s)
		return nil, os.ErrNotExist
	}

	c := DirCensus{Dir: dir, Threshold: filt}
	if c.NumRecorded() != 0 {
		t.Errorf("Unexpected NumRecorded(), expected 0 got %v", c.NumRecorded())
	}
	c.Add(20, key1)
	c.Add(21, key1)
	c.Add(30, key2)
	c.Add(31, key2)
	c.Add(32, key2)

	p := decoded(t, b)
	if p.Key != key2 {
		t.Errorf("Unexpected key, expected %v got %+v", key2, p)
	}
	if p.Count != 3 {
		t.Errorf("Unexpected count, expected 3 got %v", p.Count)
	}
	if c.NumRecorded() != 1 {
		t.Errorf("Unexpected NumRecorded(), expected 1 got %v", c.NumRecorded())
	}
}

func TestRemove(t *testing.T) {
	dir := "/path/foo"
	key := fakeKey(0x100)
	file := path.Join(dir, "100")
	filt := func(p Population) bool { return p.Count > 2 }

	var ok bool
	b := &closeBuffer{}
	deps.Stat = func(s string) (os.FileInfo, error) { return nil, os.ErrNotExist }
	deps.Create = func(s string) (io.ReadWriteCloser, error) {
		if s == file {
			ok = true
			return b, nil
		}
		t.Errorf("Create/Open called with unexpected filename, wanted %v got %v", file, s)
		return nil, os.ErrNotExist
	}
	deps.Open = deps.Create

	c := DirCensus{Dir: dir, Threshold: filt}
	if c.NumRecorded() != 0 {
		t.Errorf("Unexpected NumRecorded(), expected 0 got %v", c.NumRecorded())
	}
	c.Add(20, key)
	c.Add(21, key)
	c.Add(22, key)
	b.Reset()
	deps.Stat = func(s string) (os.FileInfo, error) { return fi{s}, nil }
	c.Remove(23, key)
	c.Remove(24, key)
	p, ok := c.Get(key)
	if !ok {
		t.Errorf("Get should have returned a population, but didn't")
	} else {
		if p.Count != 1 {
			t.Errorf("Pop count should be 1, got %v", p.Count)
		}
	}
	if b.Len() != 0 {
		t.Errorf("should not have any bytes written with one population count, found %d", b.Len())
	}
	c.Remove(25, key)

	p = decoded(t, b)
	if p.Key != key {
		t.Errorf("Unexpected key, expected %v got %+v", key, p)
	}
	if p.Last != 25 {
		t.Errorf("Unexpected last time, expected 25 got %v", p.Last)
	}
}
