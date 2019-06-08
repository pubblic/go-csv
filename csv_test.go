package csv

import (
	"bytes"
	"encoding/csv"
	"testing"
)

func writes(t *testing.T, records ...interface{}) *bytes.Buffer {
	buf := new(bytes.Buffer)
	w := NewWriter(csv.NewWriter(buf))
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
	actual := writes(t, []interface{}{
		"abcdef",
		"ghijkl",
		"mnopqr",
		"stuvwx",
		"yz",
	}).String()
	expected := "abcdef,ghijkl,mnopqr,stuvwx,yz\n"
	if expected != actual {
		t.Fatalf("expected %s, got %s", expected, actual)
	}

	actual = writes(t, []interface{}{
		123,
		456,
		7,
		"ab",
		"def",
	}).String()
	expected = "123,456,7,ab,def\n"
	if expected != actual {
		t.Fatalf("expected %s, got %s", expected, actual)
	}
}
