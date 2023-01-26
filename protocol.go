package ltvgo

type TypeCode uint8
type SizeCode uint8

const NopTag byte = 0xFF

const (
	Nil    TypeCode = 0
	Struct TypeCode = 1
	List   TypeCode = 2
	End    TypeCode = 3
	String TypeCode = 4
	Bool   TypeCode = 5
	U8     TypeCode = 6
	U16    TypeCode = 7
	U32    TypeCode = 8
	U64    TypeCode = 9
	I8     TypeCode = 10
	I16    TypeCode = 11
	I32    TypeCode = 12
	I64    TypeCode = 13
	F32    TypeCode = 14
	F64    TypeCode = 15
)

const (
	SizeSingle SizeCode = 0
	Size1      SizeCode = 1
	Size2      SizeCode = 2
	Size4      SizeCode = 3
	Size8      SizeCode = 4
)

var typeSizes = []int{0, 0, 0, 0, 1, 1, 1, 2, 4, 8, 1, 2, 4, 8, 4, 8}
var typeNames = []string{
	"Nil",
	"Struct",
	"List",
	"End",
	"String",
	"Bool",
	"U8",
	"U16",
	"U32",
	"U64",
	"I8",
	"I16",
	"I32",
	"I64",
	"F32",
	"F64",
}

func (t TypeCode) String() string {
	return typeNames[t]
}

func (t TypeCode) Size() int {
	return typeSizes[t]
}
