package sheets

import (
	"image"
	"reflect"
	"testing"
)

func checkAt(t *testing.T, f Sheet, x, y int, expected interface{}) {
	got := f.At(x, y)
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("At(%d, %d) is expected to return %d, but %d",
			x, y, expected, got)
	}
}

func checkSheet(t *testing.T, f Sheet, b image.Rectangle) {
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			f.Set(x, y, x+y)
		}
	}

	b1 := f.Bounds()
	if b != b1 {
		t.Fatalf("Bounds mismatched: %v, %v", b, b1)
	}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			checkAt(t, f, x, y, x+y)
		}
	}
}

func TestFixed(t *testing.T) {
	f := NewFixed(image.Rect(-31, -32, 33, 34))
	f.Set(-50, 50, 1234)
	f.Set(100, 100, 5678)
	checkSheet(t, f, f.Bounds())
}

func TestFloat(t *testing.T) {
	var f Float
	checkAt(t, &f, -500, -500, nil)
	b := f.Bounds()
	zero := image.Rectangle{}
	if b != zero {
		t.Fatalf("Zero bounds expected: %v", b)
	}
	checkSheet(t, &f, image.Rect(-31, -32, 33, 34))
	checkSheet(t, &f, image.Rect(-31, -32, 50, 100))
}
