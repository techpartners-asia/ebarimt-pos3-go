package postest_test

import "testing"

// fakeT lets Example, which has no *testing.T, use postest.New.
type fakeT struct {
	testing.TB
	cleanups []func()
}

func (f *fakeT) Helper()           {}
func (f *fakeT) Cleanup(fn func()) { f.cleanups = append(f.cleanups, fn) }

func (f *fakeT) close() {
	for _, fn := range f.cleanups {
		fn()
	}
}
