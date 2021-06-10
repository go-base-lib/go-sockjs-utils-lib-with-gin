package data

type FieldInfo struct {
	name     string
	startPos int64
	endPos   int64
	len      int64
	children map[string]*FieldInfo
}

func (this *FieldInfo) Len() int64 {
	return this.len
}

func (this *FieldInfo) Name() string {
	return this.name
}

func (this *FieldInfo) StartPos() int64 {
	return this.startPos
}

func (this *FieldInfo) EndPos() int64 {
	return this.endPos
}

func (this *FieldInfo) Children() map[string]*FieldInfo {
	return this.children
}
