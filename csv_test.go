package csv

import (
	"bytes"
	"testing"
)

func writes(t *testing.T, records ...interface{}) *bytes.Buffer {
	buf := new(bytes.Buffer)
	w := NewWriter(buf)
	check(t, w.WriteAll(records))
	check(t, w.Flush())
	return buf
}

func check(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

func TestWriter(t *testing.T) {
	const expected = "abcdef,ghijkl,mnopqr,stuvwx,yz\n"
	actual := writes(t, []string{
		"abcdef",
		"ghijkl",
		"mnopqr",
		"stuvwx",
		"yz",
	}).String()
	if expected != actual {
		t.Fatalf("expected %s, got %s", expected, actual)
	}
}
