package sheets

import "strconv"

type cake struct {
	orig []interface{}
	off  int
	len  int
	lcap int
}

func (c cake) lo() int {
	return c.off
}

func (c cake) hi() int {
	return c.off + c.len
}

func (c cake) slice() []interface{} {
	return c.orig[c.lcap:]
}

func (c cake) rcap() int {
	return cap(c.orig) - len(c.orig)
}

func (c cake) offset(i int) int {
	return i - c.off + c.lcap
}

func (c cake) index(i int) interface{} {
	if i < c.off || i >= c.off+c.len {
		panic("index out of range")
	}
	return c.orig[c.offset(i)]
}

func (c cake) assign(i int, v interface{}) {
	if i < c.off || i >= c.off+c.len {
		panic("index out of range")
	}
	c.orig[c.offset(i)] = v
}

func (c cake) slice2(i, j int) cake {
	if i > j {
		panic("slice bounds out of range")
	}
	return cake{
		orig: c.orig[:c.offset(j)],
		off:  i,
		len:  j - i,
		lcap: c.offset(i),
	}
}

func (c cake) lgrow(n int) cake {
	if n <= c.lcap {
		c.off = c.off - n
		c.len = c.len + n
		c.lcap = c.lcap - n
		return c
	}
	lcap := nextcap(n)
	buf := make([]interface{}, lcap+len(c.orig), lcap+cap(c.orig)-c.lcap)
	copy(buf[lcap:], c.orig)
	return cake{
		orig: buf,
		off:  c.off - n,
		len:  c.len + n,
		lcap: lcap - n,
	}
}

func (c cake) rgrow(n int) cake {
	if n <= c.rcap() {
		c.orig = c.orig[:len(c.orig)+n]
		c.len += n
		return c
	}
	rcap := nextcap(n)
	buf := make([]interface{}, len(c.orig)+n, len(c.orig)+rcap)
	copy(buf, c.orig)
	c.orig = buf
	c.len += n
	return c
}

func (c cake) xgrow(i int) cake {
	if i < c.off {
		return c.lgrow(c.off - i)
	}
	if i >= c.off+c.len {
		return c.rgrow(i - c.off - c.len + 1)
	}
	return c
}

func makeCake(orig []interface{}, lcap, off int) cake {
	return cake{
		orig: orig,
		off:  off,
		len:  len(orig) - lcap,
		lcap: lcap,
	}
}

func nextcap(n int) int {
	a := msb(n) << 1
	b := ((n << 1) + n) >> 1
	return min(a, b)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func msb(n int) int {
	for i := 0; i < (strconv.IntSize >> 1); i++ {
		n |= n >> (1 << uint(i))
	}
	return n ^ (n >> 1)
}
