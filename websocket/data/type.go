package data

type FieldType int

const (
	FieldTypeString FieldType = iota + 1
	FieldTypeInteger
	FieldTypeDouble
	FieldTypeBool
	FieldTypeList
	FieldTypeStruct
)
