// Unmarshal function for LiteVector encoding.
// This is an adaptation of the Go standard library's
// JSON Unmarshal function
// https://go.dev/src/encoding/json/decode.go

// Original
// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style

package ltvgo

import (
	"encoding"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
)

func Unmarshal(data []byte, v any) error {
	var d decodeState
	d.init(data)
	return d.unmarshal(v)
}

// Unmarshaler is the interface implemented by types
// that can unmarshal a LiteVector description of themselves.
// UnmarshalLTV must copy whatever it wishes to retain after returning.
type Unmarshaler interface {
	UnmarshalLTV([]byte) error
}

// An UnmarshalTypeError describes a LiteVector value that was
// not appropriate for a value of a specific Go type.
type UnmarshalTypeError struct {
	Desc   LtvDesc      // LiteVector descriptor
	GoType reflect.Type // Go value type it could not be assigned to
	Struct string       // name of the struct type containing the field
	Field  string       // the full path from root node to the field
}

func (e *UnmarshalTypeError) Error() string {
	if e.Struct != "" || e.Field != "" {
		return "ltv: cannot unmarshal " + e.Desc.TypeCode.String() + " into Go struct field " + e.Struct + "." + e.Field + " of type " + e.GoType.String()
	}
	return "ltv: cannot unmarshal " + e.Desc.TypeCode.String() + " into Go value of type " + e.GoType.String()
}

// An InvalidUnmarshalError describes an invalid argument passed to Unmarshal.
// (The argument to Unmarshal must be a non-nil pointer.)
type InvalidUnmarshalError struct {
	Type reflect.Type
}

func (e *InvalidUnmarshalError) Error() string {
	if e.Type == nil {
		return "ltv: Unmarshal(nil)"
	}

	if e.Type.Kind() != reflect.Pointer {
		return "ltv: Unmarshal(non-pointer " + e.Type.String() + ")"
	}
	return "ltv: Unmarshal(nil " + e.Type.String() + ")"
}

// An errorContext provides context for type errors during decoding.
type errorContext struct {
	Struct     reflect.Type
	FieldStack []string
}

type decodeState struct {
	decoder      Decoder
	errorContext *errorContext
	savedError   error
}

func (d *decodeState) init(data []byte) *decodeState {
	d.decoder = *NewDecoder(data)
	//d.decoder.Init(data)
	return d
}

// saveError saves the first err it is called with,
// for reporting at the end of the unmarshal.
func (d *decodeState) saveError(err error) {
	if d.savedError == nil {
		d.savedError = d.addErrorContext(err)
	}
}

// addErrorContext returns a new error enhanced with information from d.errorContext
func (d *decodeState) addErrorContext(err error) error {
	if d.errorContext != nil && (d.errorContext.Struct != nil || len(d.errorContext.FieldStack) > 0) {
		switch err := err.(type) {
		case *UnmarshalTypeError:
			err.Struct = d.errorContext.Struct.Name()
			err.Field = strings.Join(d.errorContext.FieldStack, ".")
		}
	}
	return err
}

func (d *decodeState) unmarshal(v any) error {

	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return &InvalidUnmarshalError{reflect.TypeOf(v)}
	}

	desc, err := d.decoder.Next()
	if err != nil {
		return d.addErrorContext(err)
	}

	err = d.value(desc, rv)
	if err != nil {
		return d.addErrorContext(err)
	}

	return d.savedError
}

func (d *decodeState) skip(desc LtvDesc) {
	err := d.decoder.Skip(desc)
	if err != nil {
		d.saveError(err)
	}
}

func (d *decodeState) value(desc LtvDesc, v reflect.Value) error {
	switch desc.TypeCode {
	case Struct:
		if v.IsValid() {
			if err := d.structure(desc, v); err != nil {
				return err
			}
		} else {
			d.skip(desc)
		}

	case List:
		if v.IsValid() {
			if err := d.list(desc, v); err != nil {
				return err
			}
		} else {
			d.skip(desc)
		}

	default:
		if v.IsValid() {
			if err := d.storeValue(desc, v); err != nil {
				return err
			}
		}
	}

	return nil
}

var textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()

func (d *decodeState) structure(desc LtvDesc, v reflect.Value) error {

	// Check for unmarshaler.
	u, _, pv := indirect(v, false)

	// Use UnmarshalLTV
	if u != nil {
		start := desc.Offset
		d.decoder.Skip(desc)
		end := d.decoder.pos
		return u.UnmarshalLTV(d.decoder.buf[start:end])
	}

	v = pv
	t := v.Type()

	// Decoding into any interface? Switch to non-reflect code.
	if v.Kind() == reflect.Interface && v.NumMethod() == 0 {
		value, err := d.decoder.ReadValue(desc)
		if err != nil {
			return err
		}

		v.Set(reflect.ValueOf(value))
		return nil
	}

	var fields structFields

	// Check type of target:
	//   struct or
	//   map[T1]T2 where T1 is string, an integer type,
	//             or an encoding.TextUnmarshaler
	switch v.Kind() {
	case reflect.Map:
		// Map key must either have string kind, have an integer kind,
		// or be an encoding.TextUnmarshaler.
		switch t.Key().Kind() {
		case reflect.String,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		default:
			if !reflect.PointerTo(t.Key()).Implements(textUnmarshalerType) {
				d.saveError(&UnmarshalTypeError{Desc: desc, GoType: v.Type()})
				d.skip(desc)
				return nil
			}
		}
		if v.IsNil() {
			v.Set(reflect.MakeMap(t))
		}
	case reflect.Struct:
		fields = cachedTypeFields(t)
		// ok
	default:
		d.saveError(&UnmarshalTypeError{Desc: desc, GoType: v.Type()})
		d.skip(desc)
		return nil
	}

	var mapElem reflect.Value
	var origErrorContext errorContext
	if d.errorContext != nil {
		origErrorContext = *d.errorContext
	}

	for {

		desc, err := d.decoder.Next()
		if err != nil {
			return err
		}

		if desc.TypeCode == End {
			break
		}

		// Read key and value
		keyVal, err := d.decoder.ReadValue(desc)
		if err != nil {
			return err
		}

		key := keyVal.(string)

		// Figure out field corresponding to key.
		var subv reflect.Value

		if v.Kind() == reflect.Map {
			elemType := t.Elem()
			if !mapElem.IsValid() {
				mapElem = reflect.New(elemType).Elem()
			} else {
				mapElem.Set(reflect.Zero(elemType))
			}
			subv = mapElem
		} else {
			var f *field
			if i, ok := fields.nameIndex[key]; ok {
				// Found an exact name match.
				f = &fields.list[i]
			} else {
				// Fall back to the expensive case-insensitive
				// linear search.
				for i := range fields.list {
					ff := &fields.list[i]
					if ff.equalFold(ff.nameBytes, []byte(key)) {
						f = ff
						break
					}
				}
			}
			if f != nil {
				subv = v
				for _, i := range f.index {
					if subv.Kind() == reflect.Pointer {
						if subv.IsNil() {
							// If a struct embeds a pointer to an unexported type,
							// it is not possible to set a newly allocated value
							// since the field is unexported.
							//
							// See https://golang.org/issue/21357
							if !subv.CanSet() {
								d.saveError(fmt.Errorf("ltv: cannot set embedded pointer to unexported struct: %v", subv.Type().Elem()))
								// Invalidate subv to ensure d.value(subv) skips over
								// the LTV value without assigning it to subv.
								subv = reflect.Value{}
								break
							}
							subv.Set(reflect.New(subv.Type().Elem()))
						}
						subv = subv.Elem()
					}
					subv = subv.Field(i)
				}
				if d.errorContext == nil {
					d.errorContext = new(errorContext)
				}
				d.errorContext.FieldStack = append(d.errorContext.FieldStack, f.name)
				d.errorContext.Struct = t
			} else {
				d.saveError(fmt.Errorf("ltv: unknown field %q", key))
			}
		}

		// Read the value element
		desc, err = d.decoder.Next()
		if err != nil {
			return d.addErrorContext(err)
		}

		if err := d.value(desc, subv); err != nil {
			return err
		}

		// Write value back to map;
		// if using struct, subv points into struct already.
		if v.Kind() == reflect.Map {
			kt := t.Key()
			var kv reflect.Value
			switch {
			case reflect.PointerTo(kt).Implements(textUnmarshalerType):
				kv = reflect.New(kt)
				// if err := d.storeValue(valueDesc, kv); err != nil {
				// 	return err
				// }
				kv = kv.Elem()
			case kt.Kind() == reflect.String:
				kv = reflect.ValueOf(key).Convert(kt)
			default:
				switch kt.Kind() {
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					n, err := strconv.ParseInt(key, 10, 64)
					if err != nil || reflect.Zero(kt).OverflowInt(n) {
						d.saveError(&UnmarshalTypeError{Desc: desc, GoType: v.Type()})
						break
					}
					kv = reflect.ValueOf(n).Convert(kt)
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
					n, err := strconv.ParseUint(key, 10, 64)
					if err != nil || reflect.Zero(kt).OverflowUint(n) {
						d.saveError(&UnmarshalTypeError{Desc: desc, GoType: v.Type()})
						break
					}
					kv = reflect.ValueOf(n).Convert(kt)
				default:
					panic("ltv: Unexpected key type") // should never occur
				}
			}
			if kv.IsValid() {
				v.SetMapIndex(kv, subv)
			}
		}

		if d.errorContext != nil {
			// Reset errorContext to its original state.
			// Keep the same underlying array for FieldStack, to reuse the
			// space and avoid unnecessary allocs.
			d.errorContext.FieldStack = d.errorContext.FieldStack[:len(origErrorContext.FieldStack)]
			d.errorContext.Struct = origErrorContext.Struct
		}
	}

	return nil
}

func (d *decodeState) list(desc LtvDesc, v reflect.Value) error {

	// Check for unmarshaler.
	u, ut, pv := indirect(v, false)

	// Use UnmarshalLTV
	if u != nil {
		start := desc.Offset
		d.decoder.Skip(desc)
		end := d.decoder.pos
		return u.UnmarshalLTV(d.decoder.buf[start:end])
	}

	if ut != nil {
		d.saveError(&UnmarshalTypeError{Desc: desc, GoType: v.Type()})
		d.skip(desc)
		return nil
	}

	v = pv

	// Check type of target.
	switch v.Kind() {
	case reflect.Interface:
		if v.NumMethod() == 0 {
			// Decoding into nil interface? Switch to non-reflect code.
			ai, err := d.decoder.readList()
			if err != nil {
				return err
			}
			v.Set(reflect.ValueOf(ai))
			return nil
		}
		// Otherwise it's invalid.
		fallthrough
	default:
		d.saveError(&UnmarshalTypeError{Desc: desc, GoType: v.Type()})
		d.skip(desc)
		return nil
	case reflect.Array, reflect.Slice:
		break
	}

	i := 0
	for {

		// Get element of array, growing if necessary.
		if v.Kind() == reflect.Slice {
			// Grow slice if necessary
			if i >= v.Cap() {
				newcap := v.Cap() + v.Cap()/2
				if newcap < 4 {
					newcap = 4
				}
				newv := reflect.MakeSlice(v.Type(), v.Len(), newcap)
				reflect.Copy(newv, v)
				v.Set(newv)
			}
			if i >= v.Len() {
				v.SetLen(i + 1)
			}
		}

		desc, err := d.decoder.Next()
		if err != nil {
			return err
		}

		if desc.TypeCode == End {
			break
		}

		if i < v.Len() {
			// Decode into element.
			if err := d.value(desc, v.Index(i)); err != nil {
				return err
			}
		} else {
			// Ran out of fixed array: skip.
			if err := d.value(desc, reflect.Value{}); err != nil {
				return err
			}
		}
		i++
	}

	if i < v.Len() {
		if v.Kind() == reflect.Array {
			// Array. Zero the rest.
			z := reflect.Zero(v.Type().Elem())
			for ; i < v.Len(); i++ {
				v.Index(i).Set(z)
			}
		} else {
			v.SetLen(i)
		}
	}

	if i == 0 && v.Kind() == reflect.Slice {
		v.Set(reflect.MakeSlice(v.Type(), 0, 0))
	}

	return nil
}

func (d *decodeState) storeValue(desc LtvDesc, v reflect.Value) error {

	u, ut, pv := indirect(v, desc.TypeCode == Nil)

	// UnmarshalLTV
	if u != nil {

		// Skip our decoder over the next value, and give the buffer to the
		// UnmarshalLTV function to handle
		start := desc.Offset
		d.decoder.Skip(desc)
		end := d.decoder.pos

		return u.UnmarshalLTV(d.decoder.buf[start:end])
	}

	value, err := d.decoder.ReadValue(desc)
	if err != nil {
		return err
	}

	// UnmarshalText
	if ut != nil && desc.TypeCode == String {
		return ut.UnmarshalText([]byte(value.(string)))
	}

	v = pv

	switch desc.TypeCode {
	default:
		d.saveError(&UnmarshalTypeError{Desc: desc, GoType: v.Type()})
	case Nil:
		switch v.Kind() {
		case reflect.Interface, reflect.Pointer, reflect.Map, reflect.Slice:
			v.Set(reflect.Zero(v.Type()))
			// otherwise, ignore null
		}
	case String:
		// Vector
		switch v.Kind() {
		default:
			d.saveError(&UnmarshalTypeError{Desc: desc, GoType: v.Type()})
		case reflect.String:
			v.SetString(value.(string))
		case reflect.Interface:
			if v.NumMethod() == 0 {
				v.Set(reflect.ValueOf(value))
			} else {
				d.saveError(&UnmarshalTypeError{Desc: desc, GoType: v.Type()})
			}
		}

	case Bool:
		d.setElement(desc, v, value, reflect.Bool)

	case U8:
		if desc.SizeCode == SizeSingle {
			d.setSingleUInt(desc, v, uint64(value.(uint8)))
		} else {
			d.setElement(desc, v, value, reflect.Uint8)
		}
	case U16:
		if desc.SizeCode == SizeSingle {
			d.setSingleUInt(desc, v, uint64(value.(uint16)))
		} else {
			d.setElement(desc, v, value, reflect.Uint16)
		}
	case U32:
		if desc.SizeCode == SizeSingle {
			d.setSingleUInt(desc, v, uint64(value.(uint32)))
		} else {
			d.setElement(desc, v, value, reflect.Uint32)
		}
	case U64:
		if desc.SizeCode == SizeSingle {
			d.setSingleUInt(desc, v, uint64(value.(uint64)))
		} else {
			d.setElement(desc, v, value, reflect.Uint64)
		}

	case I8:
		if desc.SizeCode == SizeSingle {
			d.setSingleInt(desc, v, int64(value.(int8)))
		} else {
			d.setElement(desc, v, value, reflect.Int8)
		}
	case I16:
		if desc.SizeCode == SizeSingle {
			d.setSingleInt(desc, v, int64(value.(int16)))
		} else {
			d.setElement(desc, v, value, reflect.Int16)
		}
	case I32:
		if desc.SizeCode == SizeSingle {
			d.setSingleInt(desc, v, int64(value.(int32)))
		} else {
			d.setElement(desc, v, value, reflect.Int32)
		}
	case I64:
		if desc.SizeCode == SizeSingle {
			d.setSingleInt(desc, v, int64(value.(int64)))
		} else {
			d.setElement(desc, v, value, reflect.Int64)
		}
	case F32:
		if desc.SizeCode == SizeSingle {
			switch v.Kind() {
			default:
				d.saveError(&UnmarshalTypeError{Desc: desc, GoType: v.Type()})
			case reflect.Float32, reflect.Float64:
				v.SetFloat(float64(value.(float32)))
			case reflect.Interface:
				if v.NumMethod() == 0 {
					v.Set(reflect.ValueOf(value))
				} else {
					d.saveError(&UnmarshalTypeError{Desc: desc, GoType: v.Type()})
				}
			}
		} else {
			d.setElement(desc, v, value, reflect.Int64)
		}
	case F64:
		if desc.SizeCode == SizeSingle {
			switch v.Kind() {
			default:
				d.saveError(&UnmarshalTypeError{Desc: desc, GoType: v.Type()})
			case reflect.Float32, reflect.Float64:
				v.SetFloat(value.(float64))
			case reflect.Interface:
				if v.NumMethod() == 0 {
					v.Set(reflect.ValueOf(value))
				} else {
					d.saveError(&UnmarshalTypeError{Desc: desc, GoType: v.Type()})
				}
			}
		} else {
			d.setElement(desc, v, value, reflect.Int64)
		}
	}

	return nil
}

// Check whether a source signed integer value will overflow a destination integer variable.
func checkSignedOverflow(dst reflect.Value, src int64) bool {
	switch dst.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if src < 0 {
			return true
		}
		return dst.OverflowUint(uint64(src))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return dst.OverflowInt(src)
	}

	panic("invalid destination type")
}

// Check whether a source unsigned integer value will overflow a destination integer variable.
func checkUnsignedOverflow(dst reflect.Value, src uint64) bool {
	switch dst.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return dst.OverflowUint(src)
	case reflect.Int:
		return src > math.MaxInt
	case reflect.Int8:
		return src > math.MaxInt8
	case reflect.Int16:
		return src > math.MaxInt16
	case reflect.Int32:
		return src > math.MaxInt32
	case reflect.Int64:
		return src > math.MaxInt64
	}

	panic("invalid destination type")
}

func (d *decodeState) setSingleInt(desc LtvDesc, v reflect.Value, value int64) {
	switch v.Kind() {
	default:
		d.saveError(&UnmarshalTypeError{Desc: desc, GoType: v.Type()})
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if checkSignedOverflow(v, value) {
			d.saveError(&UnmarshalTypeError{Desc: desc, GoType: v.Type()})
			return
		}
		v.SetUint(uint64(value))

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if checkSignedOverflow(v, value) {
			d.saveError(&UnmarshalTypeError{Desc: desc, GoType: v.Type()})
			return
		}
		v.SetInt(value)

	case reflect.Interface:
		if v.NumMethod() == 0 {
			// Try to get to a native 'int' if it'll fit
			if value < math.MinInt || value > math.MaxInt {
				v.Set(reflect.ValueOf(value))
			} else {
				v.Set(reflect.ValueOf(int(value)))
			}
		} else {
			d.saveError(&UnmarshalTypeError{Desc: desc, GoType: v.Type()})
		}
	}
}

func (d *decodeState) setSingleUInt(desc LtvDesc, v reflect.Value, value uint64) {
	switch v.Kind() {
	default:
		d.saveError(&UnmarshalTypeError{Desc: desc, GoType: v.Type()})
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if checkUnsignedOverflow(v, value) {
			d.saveError(&UnmarshalTypeError{Desc: desc, GoType: v.Type()})
			return
		}
		v.SetUint(value)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if checkUnsignedOverflow(v, value) {
			d.saveError(&UnmarshalTypeError{Desc: desc, GoType: v.Type()})
			return
		}
		v.SetInt(int64(value))

	case reflect.Interface:
		if v.NumMethod() == 0 {
			// Try to get to a native 'int' if it'll fit
			if value > math.MaxInt {
				v.Set(reflect.ValueOf(value))
			} else {
				v.Set(reflect.ValueOf(int(value)))
			}
		} else {
			d.saveError(&UnmarshalTypeError{Desc: desc, GoType: v.Type()})
		}
	}
}

// A slow, messy, allocating process of shoehorning a integer vector
// into a Go integer vector of another type.
// This could probably be optimized quite a bit. The best way: don't do it.
func intVectorConv(dst reflect.Value, src any) any {
	srcType := reflect.TypeOf(src)

	srcSlice := reflect.ValueOf(src)
	dstSlice := reflect.MakeSlice(reflect.SliceOf(dst.Type().Elem()), srcSlice.Len(), srcSlice.Len())

	if srcType.Elem().Kind() >= reflect.Int && srcType.Elem().Kind() <= reflect.Int64 {
		for i := 0; i < srcSlice.Len(); i++ {

			srcVal := srcSlice.Index(i).Int()
			if checkSignedOverflow(dstSlice.Index(0), srcVal) {
				return nil
			}

			switch k := dstSlice.Index(i).Kind(); k {
			default:
				panic("Unexpected slice type")
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				dstSlice.Index(i).SetInt(srcVal)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				dstSlice.Index(i).SetUint(uint64(srcVal))
			}
		}
	} else {
		for i := 0; i < srcSlice.Len(); i++ {
			srcVal := srcSlice.Index(i).Uint()
			if checkUnsignedOverflow(dstSlice.Index(0), srcVal) {
				return nil
			}

			switch k := dstSlice.Index(i).Kind(); k {
			default:
				panic("Unexpected slice type")
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				dstSlice.Index(i).SetInt(int64(srcVal))
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				dstSlice.Index(i).SetUint(uint64(srcVal))
			}
		}
	}

	return dstSlice.Interface()
}

// Convert two floating point number arrays to each other
func floatVectorConv(dst reflect.Value, src any) any {
	// We're only here if dst and src are vectors of floating point numbers,
	// and they are of different types. There are only two combinations:
	// []float64 => []float32  or
	// []float32 => []float64

	if dst.Type().Elem().Kind() == reflect.Float32 {
		srcSlice := src.([]float64)
		dstSlice := make([]float32, len(srcSlice))
		for i := 0; i < len(srcSlice); i++ {
			dstSlice[i] = float32(srcSlice[i])
		}
		return dstSlice
	} else {
		srcSlice := src.([]float32)
		dstSlice := make([]float64, len(srcSlice))
		for i := 0; i < len(srcSlice); i++ {
			dstSlice[i] = float64(srcSlice[i])
		}
		return dstSlice
	}
}

func vectorConv(dst reflect.Value, src any) any {

	srcType := reflect.TypeOf(src)

	// Check compatibility
	switch dst.Kind() {
	default:
		return nil
	case reflect.Slice:

		// Make sure the source is a vector
		if srcType.Kind() != reflect.Slice {
			return nil
		}

		dstKind := dst.Type().Elem().Kind()
		srcKind := srcType.Elem().Kind()

		// If they're the same kind, just use the original source
		if dstKind == srcKind {
			return src
		}

		// Potentially convert integer slices
		if (dstKind >= reflect.Int && dstKind <= reflect.Uint64) &&
			(srcKind >= reflect.Int || srcKind <= reflect.Uint64) {
			return intVectorConv(dst, src)
		}

		// Convert floating point slices
		if (dstKind == reflect.Float32 || dstKind == reflect.Float64) &&
			(srcKind == reflect.Float32 || srcKind == reflect.Float64) {
			return floatVectorConv(dst, src)
		}

		// Incompatible
		return nil

	case reflect.Interface:
		// The generic interface can hold it.
		if dst.NumMethod() == 0 {
			return src
		} else {
			return nil
		}
	}

}

func (d *decodeState) setElement(desc LtvDesc, v reflect.Value, value any, kind reflect.Kind) {

	if desc.SizeCode == SizeSingle {
		// Single
		switch v.Kind() {
		default:
			d.saveError(&UnmarshalTypeError{Desc: desc, GoType: v.Type()})
		case kind:
			v.Set(reflect.ValueOf(value))
		case reflect.Interface:
			if v.NumMethod() == 0 {
				v.Set(reflect.ValueOf(value))
			} else {
				d.saveError(&UnmarshalTypeError{Desc: desc, GoType: v.Type()})
			}
		}
	} else {
		// Vector
		vec := vectorConv(v, value)
		if vec != nil {
			v.Set(reflect.ValueOf(vec))
		} else {
			d.saveError(&UnmarshalTypeError{Desc: desc, GoType: v.Type()})
		}

		// switch v.Kind() {
		// default:
		// case reflect.Slice:
		// 	if v.Type().Elem().Kind() != kind {
		// 		d.saveError(&UnmarshalTypeError{Desc: desc, GoType: v.Type()})
		// 	}
		// 	v.Set(reflect.ValueOf(value))
		// case reflect.Interface:
		// 	if v.NumMethod() == 0 {
		// 		v.Set(reflect.ValueOf(value))
		// 	} else {
		// 		d.saveError(&UnmarshalTypeError{Desc: desc, GoType: v.Type()})
		// 	}
		// }
	}
}

// indirect walks down v allocating pointers as needed,
// until it gets to a non-pointer.
// If it encounters an Unmarshaler, indirect stops and returns that.
// If decodingNull is true, indirect stops at the first settable pointer so it
// can be set to nil.
func indirect(v reflect.Value, decodingNull bool) (Unmarshaler, encoding.TextUnmarshaler, reflect.Value) {
	// Issue #24153 indicates that it is generally not a guaranteed property
	// that you may round-trip a reflect.Value by calling Value.Addr().Elem()
	// and expect the value to still be settable for values derived from
	// unexported embedded struct fields.
	//
	// The logic below effectively does this when it first addresses the value
	// (to satisfy possible pointer methods) and continues to dereference
	// subsequent pointers as necessary.
	//
	// After the first round-trip, we set v back to the original value to
	// preserve the original RW flags contained in reflect.Value.
	v0 := v
	haveAddr := false

	// If v is a named type and is addressable,
	// start with its address, so that if the type has pointer methods,
	// we find them.
	if v.Kind() != reflect.Pointer && v.Type().Name() != "" && v.CanAddr() {
		haveAddr = true
		v = v.Addr()
	}
	for {
		// Load value from interface, but only if the result will be
		// usefully addressable.
		if v.Kind() == reflect.Interface && !v.IsNil() {
			e := v.Elem()
			if e.Kind() == reflect.Pointer && !e.IsNil() && (!decodingNull || e.Elem().Kind() == reflect.Pointer) {
				haveAddr = false
				v = e
				continue
			}
		}

		if v.Kind() != reflect.Pointer {
			break
		}

		if decodingNull && v.CanSet() {
			break
		}

		// Prevent infinite loop if v is an interface pointing to its own address:
		//     var v interface{}
		//     v = &v
		if v.Elem().Kind() == reflect.Interface && v.Elem().Elem() == v {
			v = v.Elem()
			break
		}
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		if v.Type().NumMethod() > 0 && v.CanInterface() {
			if u, ok := v.Interface().(Unmarshaler); ok {
				return u, nil, reflect.Value{}
			}
			if !decodingNull {
				if u, ok := v.Interface().(encoding.TextUnmarshaler); ok {
					return nil, u, reflect.Value{}
				}
			}
		}

		if haveAddr {
			v = v0 // restore original value after round-trip Value.Addr().Elem()
			haveAddr = false
		} else {
			v = v.Elem()
		}
	}
	return nil, nil, v
}
