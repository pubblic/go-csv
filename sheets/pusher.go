package sheets

import "image"

type Pusher interface {
	Push(val interface{})
}

type VecPusher struct {
	Sheet Sheet
	Vec   image.Point
	Dir   image.Point
}

func (p *VecPusher) Push(val interface{}) {
	p.Sheet.Set(p.Vec.X, p.Vec.Y, val)
	p.Vec = p.Vec.Add(p.Dir)
}

func (p *VecPusher) SetDir(x, y int) *VecPusher {
	p.Dir = image.Pt(x, y)
	return p
}

func (p *VecPusher) SetVec(x, y int) *VecPusher {
	p.Vec = image.Pt(x, y)
	return p
}

func NewVecPusher(sheet Sheet) *VecPusher {
	return &VecPusher{
		Sheet: sheet,
	}
}
