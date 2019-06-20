package sheets

import (
	"image"
)

type Sheet interface {
	Bounds() image.Rectangle
	At(x, y int) interface{}
	Set(x, y int, val interface{})
}

type Pusher interface {
	Push(val interface{})
}

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
