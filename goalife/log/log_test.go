package log

import "bytes"
import "io/ioutil"
import "log"
import "testing"

func runNullVsDiscard(b *testing.B, l Logger) {
	s := struct {
		a int
		b string
	}{
		42,
		"foo",
	}
	for i := 0; i < b.N; i++ {
		s.a += i
		l.Printf("%#v %d %s\n", s, s.a, s.b)
	}
}

func BenchmarkKeep(b *testing.B) {
	var buf bytes.Buffer
	runNullVsDiscard(b, log.New(&buf, "", log.LstdFlags|log.Lshortfile))
}

func BenchmarkDiscard(b *testing.B) {
	runNullVsDiscard(b, log.New(ioutil.Discard, "", log.LstdFlags|log.Lshortfile))
}

func BenchmarkNull(b *testing.B) {
	runNullVsDiscard(b, Null())
}
