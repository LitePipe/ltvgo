package bench

import (
	"math"
	"testing"

	ltv "github.com/ThadThompson/ltvgo"
)

type SmallObject struct {
	A int
	B float64
	C string
}

type MediumObject struct {
	Name         string
	Rank         string
	SerialNumber uint64
	Embedded     SmallObject
	AList        []any
	Anum         int32
	AnotherNum   uint64
}

type LargeObject struct {
	Name         string
	Width        int
	Height       int
	ImgData      []byte
	Measurements []float64
}

type TestObject interface {
	SmallObject | MediumObject | LargeObject
}

var smallData = SmallObject{
	A: 123,
	B: 456.789,
	C: "Yay",
}

var medData = MediumObject{
	Name:         "Captain Awesome",
	Rank:         "Private",
	SerialNumber: 123456789,
	Embedded:     smallData,
	AList:        []any{"Chimi", "Changa", 55},
	Anum:         -123456789,
	AnotherNum:   math.MaxUint64,
}

var largeData = createLargeData()

func createLargeData() (l LargeObject) {
	l.Name = "All the Data"
	l.Width = 640
	l.Height = 480
	l.ImgData = make([]byte, l.Width*l.Height*3)
	l.Measurements = make([]float64, l.Width*l.Height)

	for h := 0; h < l.Height; h++ {
		for w := 0; w < l.Width; w++ {
			l.ImgData[h*l.Width+w+0] = uint8(h + w + 0)
			l.ImgData[h*l.Width+w+1] = uint8(h + w + 1)
			l.ImgData[h*l.Width+w+2] = uint8(h + w + 2)
			l.Measurements[h*l.Width+w] = float64(h*w) * 0.123
		}
	}

	return l
}

func benchMarshalLtv(value any, b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := ltv.Marshal(value)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func benchUnmarshalLtv[T TestObject](value T, b *testing.B) {
	data, err := ltv.Marshal(value)
	if err != nil {
		b.Fatal(err)
	}

	var ret T
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := ltv.Unmarshal(data, &ret)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLtvMarshalSmall(b *testing.B)  { benchMarshalLtv(smallData, b) }
func BenchmarkLtvMarshalMedium(b *testing.B) { benchMarshalLtv(medData, b) }
func BenchmarkLtvMarshalLarge(b *testing.B)  { benchMarshalLtv(largeData, b) }

func BenchmarkLtvUnmarshalSmall(b *testing.B)  { benchUnmarshalLtv(smallData, b) }
func BenchmarkLtvUnmarshalMedium(b *testing.B) { benchUnmarshalLtv(medData, b) }
func BenchmarkLtvUnmarshalLarge(b *testing.B)  { benchUnmarshalLtv(largeData, b) }

// Comparison Benchmarks

// func benchMarshalJson(value any, b *testing.B) {
// 	for i := 0; i < b.N; i++ {
// 		_, err := json.Marshal(value)
// 		if err != nil {
// 			b.Fatal(err)
// 		}
// 	}
// }

// func benchUnmarshalJson[T TestObject](value T, b *testing.B) {
// 	data, err := json.Marshal(value)
// 	if err != nil {
// 		b.Fatal(err)
// 	}

// 	var ret T
// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		err := json.Unmarshal(data, &ret)
// 		if err != nil {
// 			b.Fatal(err)
// 		}
// 	}
// }

// func BenchmarkJsonMarshalSmall(b *testing.B)  { benchMarshalJson(smallData, b) }
// func BenchmarkJsonMarshalMedium(b *testing.B) { benchMarshalJson(medData, b) }
// func BenchmarkJsonMarshalLarge(b *testing.B)  { benchMarshalJson(largeData, b) }

// func BenchmarkJsonUnmarshalSmall(b *testing.B)  { benchUnmarshalJson(smallData, b) }
// func BenchmarkJsonUnmarshalMedium(b *testing.B) { benchUnmarshalJson(medData, b) }
// func BenchmarkJsonUnmarshalLarge(b *testing.B)  { benchUnmarshalJson(largeData, b) }
