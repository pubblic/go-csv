package csv

import (
	"bytes"
	"encoding/csv"
	"reflect"
	"strings"
	"testing"
)

var csvTests = []struct {
	args []interface{}
	want string
}{
	{nil, ""},
	{
		[]interface{}{
			nil,
		},
		"\n",
	},
	{
		[]interface{}{
			[]string{"ab", "cd", "ef"},
			[]interface{}{"hi", "jk", "lm", "no", 90},
			[]string{"qr", "st"},
		},
		"ab,cd,ef\nhi,jk,lm,no,90\nqr,st\n",
	},
	{
		[]interface{}{
			[]int64{19, 23, 257},
			[]interface{}{nil, "abc", nil, 22},
			[]interface{}{stringType("hello"), nil},
		},
		"19,23,257\n,abc,,22\nhello,\n",
	},
	{
		[]interface{}{
			nil,
			[]*int{nil, nil},
			[]*string{},
			[]interface{}{43, (*int)(nil), "abc"},
		},
		"\n,\n\n43,,abc\n",
	},
}

type stringType string

func TestWriter(t *testing.T) {
	for i, tc := range csvTests {
		buf, err := bufWrite(tc.args)
		if err != nil {
			t.Fatal(err)
		}
		if buf.String() != tc.want {
			t.Fatalf("Test#%d wants %s, but got %s",
				i+1, tc.want, buf.String())
		}
	}
}

func bufWrite(args []interface{}) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	w := NewWriter(csv.NewWriter(buf))
	err := w.WriteAll(args)
	if err != nil {
		return nil, err
	}
	err = w.Flush()
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func TestReader(t *testing.T) {
	const data = "abc,32,17,,bytes\n"
	r := NewReader(csv.NewReader(strings.NewReader(data)))
	r.Reader.ReuseRecord = true

	for r.Next() {
		var (
			a string
			b int
			c uint32
			d string
			e []byte
		)
		err := r.Scan(&a, &b, &c, &d, &e)
		if err != nil {
			t.Fatal(err)
		}
		equal(t, "abc", a)
		equal(t, 32, b)
		equal(t, uint32(17), c)
		equal(t, "", d)
		equal(t, []byte("bytes"), e)
	}
	err := r.Err()
	if err != nil {
		t.Fatal(err)
	}
}

func equal(t *testing.T, want, got interface{}) {
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("wants %v, got %v", want, got)
	}
}
