package sheets

import (
	"image"
)

// Sheet 인터페이스는 시트 객체를 표현하는 데 사용합니다. 시트 객체는 Fixed
// 같이 일정한 크기를 가지거나 또는 Float 같이유동적인 크기를 가질 수 있습니다.
type Sheet interface {
	// Bounds는 시트의 크기를 담은 r image.Rectangle 객체를 반환합니다.
	Bounds() image.Rectangle

	// At은 셀 (x, y)에 입력된 값을 가져옵니다. 시트 크기를 벗어난 셀에
	// 대하여 nil 값을 반환합니다.
	At(x, y int) interface{}

	// Set은 셀 (x, y)에 값을 입력합니다. 고정된 시트 크기를 가진 경우 시트
	// 크기를 넘어선 셀에 값을 입력할 경우 에러 없이 값이 입력되지 않을 수
	// 있습니다.
	Set(x, y int, val interface{})
}

// Fixed는 고정된 크기를 가진 시트를 표현합니다.
type Fixed struct {
	Pix []interface{}

	Stride int

	Rect image.Rectangle
}

func NewFixed(r image.Rectangle) *Fixed {
	w, h := r.Dx(), r.Dy()
	buf := make([]interface{}, w*h)
	return &Fixed{buf, w, r}
}

func (p *Fixed) Bounds() image.Rectangle {
	return p.Rect
}

func (p *Fixed) At(x, y int) interface{} {
	i := p.PixOffset(x, y)
	return p.Pix[i]
}

func (p *Fixed) PixOffset(x, y int) int {
	return (y-p.Rect.Min.Y)*p.Stride + (x - p.Rect.Min.X)
}

func (p *Fixed) Set(x, y int, val interface{}) {
	if !(image.Point{x, y}.In(p.Rect)) {
		return
	}
	i := p.PixOffset(x, y)
	p.Pix[i] = val
}

// Float은 유동적인 크기를 가진 시트를 표현합니다. 시트 크기는 입력된 값에 따라
// 자동적으로 변경됩니다.
type Float struct {
	rows cake
}

func (p *Float) Bounds() image.Rectangle {
	bx, by := p.boundsX(), p.boundsY()
	return image.Rect(bx.X, by.X, bx.Y, by.Y)
}

func (p *Float) boundsX() (u image.Point) {
	orig := p.rows.slice()
	for i := range orig {
		row, ok := orig[i].(*rowVec)
		if !ok {
			continue
		}
		b := row.bounds()
		if !isZeroPoint(b) {
			u = unionPoint(u, b)
		}
	}
	return u
}

func (p *Float) boundsY() image.Point {
	return image.Point{
		p.rows.off,
		p.rows.off + p.rows.len,
	}
}

func isZeroPoint(p image.Point) bool {
	return p.X == p.Y
}

func unionPoint(a, b image.Point) image.Point {
	return image.Point{min(a.X, b.X), max(a.Y, b.Y)}
}

func max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

func (p *Float) row(i int) *rowVec {
	row, _ := p.rows.index(i).(*rowVec)
	if row == nil {
		row = new(rowVec)
		p.rows.assign(i, row)
	}
	return row
}

func (p *Float) Set(x, y int, val interface{}) {
	p.rows = p.rows.xgrow(y)
	p.row(y).set(x, val)
	p.resize()
}

func (p *Float) resize() {
	i := p.rows.lo()
	j := p.rows.hi()
	for i < j && p.row(i).isEmpty() {
		i++
	}
	for i < j && p.row(j-1).isEmpty() {
		j--
	}
	p.rows = p.rows.slice2(i, j)
}

func (p *Float) At(x, y int) interface{} {
	if y < p.rows.lo() || p.rows.hi() <= y {
		return nil
	}
	row, _ := p.rows.index(y).(*rowVec)
	return row.at(x)
}

type rowVec struct {
	vec cake
}

func (r *rowVec) bounds() image.Point {
	return image.Point{r.vec.off, r.vec.off + r.vec.len}
}

func (r *rowVec) isEmpty() bool {
	for _, v := range r.vec.slice() {
		if v != nil {
			return false
		}
	}
	return true
}

func (r *rowVec) at(x int) interface{} {
	if r == nil {
		return nil
	}
	return r.vec.index(x)
}

func (r *rowVec) set(x int, val interface{}) {
	r.vec = r.vec.xgrow(x)
	r.vec.assign(x, val)
	r.resize()
}

func (r *rowVec) resize() {
	i, j := r.vec.lo(), r.vec.hi()
	for i < j && r.vec.index(i) == nil {
		i++
	}
	for i < j && r.vec.index(j-1) == nil {
		j--
	}
	r.vec = r.vec.slice2(i, j)
}
