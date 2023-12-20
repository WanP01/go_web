package cache

type ByteView struct {
	b []byte
}

// Len for 支持 Value interface
func (v ByteView) Len() int {
	return len(v.b)
}

// ByteSlice 返回数据切片
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

// String 返回数据字符串表示
func (v ByteView) String() string {
	return string(v.b)
}

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
