package csv

import (
	"bytes"
	"encoding/csv"
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

func check(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

	}
}
