// Marshal function for LiteVector encoding.
// This is an adaptation of the Go standard library's
// JSON Marshal function
// https://go.dev/src/encoding/json/encode.go

// Original
// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ltvgo

import (
	"encoding"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

func Marshal(v any) ([]byte, error) {
	e := newEncodeState()
	defer encodeStatePool.Put(e)

	err := e.marshal(v, encOpts{})
	if err != nil {
		return nil, err
	}
	buf := append([]byte(nil), e.l.Bytes()...)

	return buf, nil
}

type Marshaler interface {
	MarshalLTV() ([]byte, error)
}

// An UnsupportedTypeError is returned by Marshal when attempting
// to encode an unsupported value type.
type UnsupportedTypeError struct {
	Type reflect.Type
}

func (e *UnsupportedTypeError) Error() string {
	return "ltv: unsupported type: " + e.Type.String()
}

// An UnsupportedValueError is returned by Marshal when attempting
// to encode an unsupported value.
type UnsupportedValueError struct {
	Value reflect.Value
	Str   string
}

func (e *UnsupportedValueError) Error() string {
	return "ltv: unsupported value: " + e.Str
}

// A MarshalerError represents an error from calling a MarshalLTV or MarshalText method.
type MarshalerError struct {
	Type       reflect.Type
	Err        error
	sourceFunc string
}

func (e *MarshalerError) Error() string {
	srcFunc := e.sourceFunc
	if srcFunc == "" {
		srcFunc = "MarshalLTV"
	}
	return "ltv: error calling " + srcFunc +
		" for type " + e.Type.String() +
		": " + e.Err.Error()
}

// Unwrap returns the underlying error.
func (e *MarshalerError) Unwrap() error { return e.Err }

// An encodeState encodes LiteVectors into a bytes.Buffer.
type encodeState struct {
	l Encoder
	//buf *bytes.Buffer // accumulated output

	// Keep track of what pointers we've seen in the current recursive call
	// path, to avoid cycles that could lead to a stack overflow. Only do
	// the relatively expensive map operations if ptrLevel is larger than
	// startDetectingCyclesAfter, so that we skip the work if we're within a
	// reasonable amount of nested pointers deep.
	ptrLevel uint
	ptrSeen  map[any]struct{}
}

const startDetectingCyclesAfter = 1000

var encodeStatePool sync.Pool

func newEncodeState() *encodeState {
	if v := encodeStatePool.Get(); v != nil {
		e := v.(*encodeState)

		e.l.Reset()
		// e.buf.Reset()
		// e.l = NewStreamEncoder(e.buf)
		if len(e.ptrSeen) > 0 {
			panic("ptrEncoder.encode should have emptied ptrSeen via defers")
		}
		e.ptrLevel = 0
		return e
	}

	return &encodeState{ptrSeen: make(map[any]struct{})}
}

// ltvError is an error wrapper type for internal use only.
// Panics with errors are wrapped in ltvError so that the top-level recover
// can distinguish intentional panics from this package.
type ltvError struct{ error }

func (e *encodeState) marshal(v any, opts encOpts) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if je, ok := r.(ltvError); ok {
				err = je.error
			} else {
				panic(r)
			}
		}
	}()
	e.reflectValue(reflect.ValueOf(v), opts)
	return nil
}

// error aborts the encoding by panicking with err wrapped in jsonError.
func (e *encodeState) error(err error) {
	panic(ltvError{err})
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Pointer:
		return v.IsNil()
	}
	return false
}

func (e *encodeState) reflectValue(v reflect.Value, opts encOpts) {
	valueEncoder(v)(e, v, opts)
}

type encOpts struct {
}

type encoderFunc func(e *encodeState, v reflect.Value, opts encOpts)

var encoderCache sync.Map

func valueEncoder(v reflect.Value) encoderFunc {
	if !v.IsValid() {
		return invalidValueEncoder
	}
	return typeEncoder(v.Type())
}

func typeEncoder(t reflect.Type) encoderFunc {
	if fi, ok := encoderCache.Load(t); ok {
		return fi.(encoderFunc)
	}

	// To deal with recursive types, populate the map with an
	// indirect func before we build it. This type waits on the
	// real func (f) to be ready and then calls it. This indirect
	// func is only used for recursive types.
	var (
		wg sync.WaitGroup
		f  encoderFunc
	)
	wg.Add(1)
	fi, loaded := encoderCache.LoadOrStore(t, encoderFunc(func(e *encodeState, v reflect.Value, opts encOpts) {
		wg.Wait()
		f(e, v, opts)
	}))
	if loaded {
		return fi.(encoderFunc)
	}

	// Compute the real encoder and replace the indirect func with it.
	f = newTypeEncoder(t, true)
	wg.Done()
	encoderCache.Store(t, f)
	return f
}

var (
	marshalerType     = reflect.TypeOf((*Marshaler)(nil)).Elem()
	textMarshalerType = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
)

// newTypeEncoder constructs an encoderFunc for a type.
// The returned encoder only checks CanAddr when allowAddr is true.
func newTypeEncoder(t reflect.Type, allowAddr bool) encoderFunc {
	// If we have a non-pointer value whose type implements
	// Marshaler with a value receiver, then we're better off taking
	// the address of the value - otherwise we end up with an
	// allocation as we cast the value to an interface.
	if t.Kind() != reflect.Pointer && allowAddr && reflect.PointerTo(t).Implements(marshalerType) {
		return newCondAddrEncoder(addrMarshalerEncoder, newTypeEncoder(t, false))
	}
	if t.Implements(marshalerType) {
		return marshalerEncoder
	}
	if t.Kind() != reflect.Pointer && allowAddr && reflect.PointerTo(t).Implements(textMarshalerType) {
		return newCondAddrEncoder(addrTextMarshalerEncoder, newTypeEncoder(t, false))
	}
	if t.Implements(textMarshalerType) {
		return textMarshalerEncoder
	}

	switch t.Kind() {
	case reflect.Bool:
		return boolEncoder
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return intEncoder
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return uintEncoder
	case reflect.Float32:
		return float32Encoder
	case reflect.Float64:
		return float64Encoder
	case reflect.String:
		return stringEncoder
	case reflect.Interface:
		return interfaceEncoder
	case reflect.Struct:
		return newStructEncoder(t)
	case reflect.Map:
		return newMapEncoder(t)
	case reflect.Slice:
		return newSliceEncoder(t)
	case reflect.Array:
		return newArrayEncoder(t)
	case reflect.Pointer:
		return newPtrEncoder(t)
	default:
		return unsupportedTypeEncoder
	}
}

func invalidValueEncoder(e *encodeState, v reflect.Value, _ encOpts) {
	e.l.WriteNil()
}

func marshalerEncoder(e *encodeState, v reflect.Value, opts encOpts) {
	if v.Kind() == reflect.Pointer && v.IsNil() {
		e.l.WriteNil()
		return
	}
	m, ok := v.Interface().(Marshaler)
	if !ok {
		e.l.WriteNil()
		return
	}
	b, err := m.MarshalLTV()
	if err == nil {
		err = Validate(b)
	}

	if err != nil {
		e.error(&MarshalerError{v.Type(), err, "MarshalLTV"})
		return
	}

	e.l.RawWrite(b)
}

func addrMarshalerEncoder(e *encodeState, v reflect.Value, opts encOpts) {
	va := v.Addr()
	if va.IsNil() {
		e.l.WriteNil()
		return
	}
	m := va.Interface().(Marshaler)
	b, err := m.MarshalLTV()
	if err == nil {
		err = Validate(b)
	}
	if err != nil {
		e.error(&MarshalerError{v.Type(), err, "MarshalLTV"})
		return
	}

	e.l.RawWrite(b)
}

func textMarshalerEncoder(e *encodeState, v reflect.Value, opts encOpts) {
	if v.Kind() == reflect.Pointer && v.IsNil() {
		e.l.WriteNil()
		return
	}
	m, ok := v.Interface().(encoding.TextMarshaler)
	if !ok {
		e.l.WriteNil()
		return
	}
	b, err := m.MarshalText()
	if err != nil {
		e.error(&MarshalerError{v.Type(), err, "MarshalText"})
	}
	e.stringBytes(b)
}

func addrTextMarshalerEncoder(e *encodeState, v reflect.Value, opts encOpts) {
	va := v.Addr()
	if va.IsNil() {
		e.l.WriteNil()
		return
	}
	m := va.Interface().(encoding.TextMarshaler)
	b, err := m.MarshalText()
	if err != nil {
		e.error(&MarshalerError{v.Type(), err, "MarshalText"})
	}
	e.stringBytes(b)
}

func boolEncoder(e *encodeState, v reflect.Value, opts encOpts) {
	e.l.WriteBool(v.Bool())
}

func intEncoder(e *encodeState, v reflect.Value, opts encOpts) {
	e.l.WriteInt(v.Int())
}

func uintEncoder(e *encodeState, v reflect.Value, opts encOpts) {
	e.l.WriteUint(v.Uint())
}

func float32Encoder(e *encodeState, v reflect.Value, opts encOpts) {
	e.l.WriteF32(float32(v.Float()))
}

func float64Encoder(e *encodeState, v reflect.Value, opts encOpts) {
	e.l.WriteF64(float64(v.Float()))
}

func stringEncoder(e *encodeState, v reflect.Value, opts encOpts) {
	e.l.WriteString(v.String())
}

func interfaceEncoder(e *encodeState, v reflect.Value, opts encOpts) {
	if v.IsNil() {
		e.l.WriteNil()
		return
	}
	e.reflectValue(v.Elem(), opts)
}

func unsupportedTypeEncoder(e *encodeState, v reflect.Value, _ encOpts) {
	e.error(&UnsupportedTypeError{v.Type()})
}

type structEncoder struct {
	fields structFields
}

type structFields struct {
	list      []field
	nameIndex map[string]int
}

func (se structEncoder) encode(e *encodeState, v reflect.Value, opts encOpts) {
	e.l.WriteStructStart()
FieldLoop:
	for i := range se.fields.list {
		f := &se.fields.list[i]

		// Find the nested struct field by following f.index.
		fv := v
		for _, i := range f.index {
			if fv.Kind() == reflect.Pointer {
				if fv.IsNil() {
					continue FieldLoop
				}
				fv = fv.Elem()
			}
			fv = fv.Field(i)
		}

		if f.omitEmpty && isEmptyValue(fv) {
			continue
		}

		e.l.WriteString(f.nameNonEsc)
		f.encoder(e, fv, opts)
	}

	e.l.WriteStructEnd()
}

func newStructEncoder(t reflect.Type) encoderFunc {
	se := structEncoder{fields: cachedTypeFields(t)}
	return se.encode
}

type mapEncoder struct {
	elemEnc encoderFunc
}

func (me mapEncoder) encode(e *encodeState, v reflect.Value, opts encOpts) {
	if v.IsNil() {
		e.l.WriteNil()
		return
	}
	if e.ptrLevel++; e.ptrLevel > startDetectingCyclesAfter {
		// We're a large number of nested ptrEncoder.encode calls deep;
		// start checking if we've run into a pointer cycle.
		ptr := v.UnsafePointer()
		if _, ok := e.ptrSeen[ptr]; ok {
			e.error(&UnsupportedValueError{v, fmt.Sprintf("encountered a cycle via %s", v.Type())})
		}
		e.ptrSeen[ptr] = struct{}{}
		defer delete(e.ptrSeen, ptr)
	}

	e.l.WriteStructStart()

	// Extract and sort the keys.
	sv := make([]reflectWithString, v.Len())
	mi := v.MapRange()
	for i := 0; mi.Next(); i++ {
		sv[i].k = mi.Key()
		sv[i].v = mi.Value()
		if err := sv[i].resolve(); err != nil {
			e.error(fmt.Errorf("ltv: encoding error for type %q: %q", v.Type().String(), err.Error()))
		}
	}
	sort.Slice(sv, func(i, j int) bool { return sv[i].ks < sv[j].ks })

	for _, kv := range sv {
		e.string(kv.ks)
		me.elemEnc(e, kv.v, opts)
	}
	e.l.WriteStructEnd()
	e.ptrLevel--
}

func newMapEncoder(t reflect.Type) encoderFunc {
	switch t.Key().Kind() {
	case reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
	default:
		if !t.Key().Implements(textMarshalerType) {
			return unsupportedTypeEncoder
		}
	}
	me := mapEncoder{typeEncoder(t.Elem())}
	return me.encode
}

func encodeBoolSlice(e *encodeState, v reflect.Value, _ encOpts) {
	if v.IsNil() {
		e.l.WriteNil()
		return
	}

	s, ok := v.Interface().([]bool)
	if !ok {
		panic("value not a []bool")
	}

	e.l.WriteBoolVec(s)
}

func encodeI8Slice(e *encodeState, v reflect.Value, _ encOpts) {
	if v.IsNil() {
		e.l.WriteNil()
		return
	}

	s, ok := v.Interface().([]int8)
	if !ok {
		panic("value not a []int8")
	}

	e.l.WriteI8Vec(s)
}

func encodeI16Slice(e *encodeState, v reflect.Value, _ encOpts) {
	if v.IsNil() {
		e.l.WriteNil()
		return
	}

	s, ok := v.Interface().([]int16)
	if !ok {
		panic("value not a []int16")
	}

	e.l.WriteI16Vec(s)
}

func encodeI32Slice(e *encodeState, v reflect.Value, _ encOpts) {
	if v.IsNil() {
		e.l.WriteNil()
		return
	}

	s, ok := v.Interface().([]int32)
	if !ok {
		panic("value not a []int32")
	}

	e.l.WriteI32Vec(s)
}

func encodeI64Slice(e *encodeState, v reflect.Value, _ encOpts) {
	if v.IsNil() {
		e.l.WriteNil()
		return
	}

	s, ok := v.Interface().([]int64)
	if !ok {
		panic("value not a []int64")
	}

	e.l.WriteI64Vec(s)
}

func encodeU8Slice(e *encodeState, v reflect.Value, _ encOpts) {
	if v.IsNil() {
		e.l.WriteNil()
		return
	}

	s, ok := v.Interface().([]uint8)
	if !ok {
		panic("value not a []uint8")
	}

	e.l.WriteU8Vec(s)
}

func encodeU16Slice(e *encodeState, v reflect.Value, _ encOpts) {
	if v.IsNil() {
		e.l.WriteNil()
		return
	}

	s, ok := v.Interface().([]uint16)
	if !ok {
		panic("value not a []uint16")
	}

	e.l.WriteU16Vec(s)
}

func encodeU32Slice(e *encodeState, v reflect.Value, _ encOpts) {
	if v.IsNil() {
		e.l.WriteNil()
		return
	}

	s, ok := v.Interface().([]uint32)
	if !ok {
		panic("value not a []uint32")
	}

	e.l.WriteU32Vec(s)
}

func encodeU64Slice(e *encodeState, v reflect.Value, _ encOpts) {
	if v.IsNil() {
		e.l.WriteNil()
		return
	}

	s, ok := v.Interface().([]uint64)
	if !ok {
		panic("value not a []uint64")
	}

	e.l.WriteU64Vec(s)
}

func encodeF32Slice(e *encodeState, v reflect.Value, _ encOpts) {
	if v.IsNil() {
		e.l.WriteNil()
		return
	}

	s, ok := v.Interface().([]float32)
	if !ok {
		panic("value not a []float32")
	}

	e.l.WriteF32Vec(s)
}

func encodeF64Slice(e *encodeState, v reflect.Value, _ encOpts) {
	if v.IsNil() {
		e.l.WriteNil()
		return
	}

	s, ok := v.Interface().([]float64)
	if !ok {
		panic("value not a []float64")
	}

	e.l.WriteF64Vec(s)
}

// sliceEncoder just wraps an arrayEncoder, checking to make sure the value isn't nil.
type sliceEncoder struct {
	arrayEnc encoderFunc
}

func (se sliceEncoder) encode(e *encodeState, v reflect.Value, opts encOpts) {
	if v.IsNil() {
		e.l.WriteNil()
		return
	}
	if e.ptrLevel++; e.ptrLevel > startDetectingCyclesAfter {
		// We're a large number of nested ptrEncoder.encode calls deep;
		// start checking if we've run into a pointer cycle.
		// Here we use a struct to memorize the pointer to the first element of the slice
		// and its length.
		ptr := struct {
			ptr interface{} // always an unsafe.Pointer, but avoids a dependency on package unsafe
			len int
		}{v.UnsafePointer(), v.Len()}
		if _, ok := e.ptrSeen[ptr]; ok {
			e.error(&UnsupportedValueError{v, fmt.Sprintf("encountered a cycle via %s", v.Type())})
		}
		e.ptrSeen[ptr] = struct{}{}
		defer delete(e.ptrSeen, ptr)
	}
	se.arrayEnc(e, v, opts)
	e.ptrLevel--
}

func newSliceEncoder(t reflect.Type) encoderFunc {

	switch t.Elem().Kind() {

	case reflect.Bool:
		return encodeBoolSlice

	case reflect.Int8:
		return encodeI8Slice
	case reflect.Int16:
		return encodeI16Slice
	case reflect.Int32:
		return encodeI32Slice
	case reflect.Int64:
		return encodeI64Slice

	case reflect.Uint8:
		return encodeU8Slice
	case reflect.Uint16:
		return encodeU16Slice
	case reflect.Uint32:
		return encodeU32Slice
	case reflect.Uint64:
		return encodeU64Slice

	case reflect.Float32:
		return encodeF32Slice
	case reflect.Float64:
		return encodeF64Slice
	}

	enc := sliceEncoder{newArrayEncoder(t)}
	return enc.encode
}

type arrayEncoder struct {
	elemEnc encoderFunc
}

func (ae arrayEncoder) encode(e *encodeState, v reflect.Value, opts encOpts) {
	e.l.WriteListStart()
	n := v.Len()
	for i := 0; i < n; i++ {
		ae.elemEnc(e, v.Index(i), opts)
	}
	e.l.WriteListEnd()
}

func newArrayEncoder(t reflect.Type) encoderFunc {
	enc := arrayEncoder{typeEncoder(t.Elem())}
	return enc.encode
}

type ptrEncoder struct {
	elemEnc encoderFunc
}

func (pe ptrEncoder) encode(e *encodeState, v reflect.Value, opts encOpts) {
	if v.IsNil() {
		e.l.WriteNil()
		return
	}
	if e.ptrLevel++; e.ptrLevel > startDetectingCyclesAfter {
		// We're a large number of nested ptrEncoder.encode calls deep;
		// start checking if we've run into a pointer cycle.
		ptr := v.Interface()
		if _, ok := e.ptrSeen[ptr]; ok {
			e.error(&UnsupportedValueError{v, fmt.Sprintf("encountered a cycle via %s", v.Type())})
		}
		e.ptrSeen[ptr] = struct{}{}
		defer delete(e.ptrSeen, ptr)
	}
	pe.elemEnc(e, v.Elem(), opts)
	e.ptrLevel--
}

func newPtrEncoder(t reflect.Type) encoderFunc {
	enc := ptrEncoder{typeEncoder(t.Elem())}
	return enc.encode
}

type condAddrEncoder struct {
	canAddrEnc, elseEnc encoderFunc
}

func (ce condAddrEncoder) encode(e *encodeState, v reflect.Value, opts encOpts) {
	if v.CanAddr() {
		ce.canAddrEnc(e, v, opts)
	} else {
		ce.elseEnc(e, v, opts)
	}
}

// newCondAddrEncoder returns an encoder that checks whether its value
// CanAddr and delegates to canAddrEnc if so, else to elseEnc.
func newCondAddrEncoder(canAddrEnc, elseEnc encoderFunc) encoderFunc {
	enc := condAddrEncoder{canAddrEnc: canAddrEnc, elseEnc: elseEnc}
	return enc.encode
}

func isValidTag(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		switch {
		case strings.ContainsRune("!#$%&()*+-./:;<=>?@[]^_{|}~ ", c):
			// Backslash and quote chars are reserved, but
			// otherwise any punctuation chars are allowed
			// in a tag name.
		case !unicode.IsLetter(c) && !unicode.IsDigit(c):
			return false
		}
	}
	return true
}

func typeByIndex(t reflect.Type, index []int) reflect.Type {
	for _, i := range index {
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		t = t.Field(i).Type
	}
	return t
}

type reflectWithString struct {
	k  reflect.Value
	v  reflect.Value
	ks string
}

func (w *reflectWithString) resolve() error {
	if w.k.Kind() == reflect.String {
		w.ks = w.k.String()
		return nil
	}
	if tm, ok := w.k.Interface().(encoding.TextMarshaler); ok {
		if w.k.Kind() == reflect.Pointer && w.k.IsNil() {
			return nil
		}
		buf, err := tm.MarshalText()
		w.ks = string(buf)
		return err
	}
	switch w.k.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		w.ks = strconv.FormatInt(w.k.Int(), 10)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		w.ks = strconv.FormatUint(w.k.Uint(), 10)
		return nil
	}
	panic("unexpected map key type")
}

// NOTE: keep in sync with stringBytes below.
func (e *encodeState) string(s string) {
	e.l.WriteString(s)
}

// NOTE: keep in sync with string above.
func (e *encodeState) stringBytes(s []byte) {
	e.l.WriteString(string(s))
}

// A field represents a single field found in a struct.
type field struct {
	name      string
	nameBytes []byte                 // []byte(name)
	equalFold func(s, t []byte) bool // bytes.EqualFold or equivalent

	nameNonEsc string // `"` + name + `":`

	tag       bool
	index     []int
	typ       reflect.Type
	omitEmpty bool
	quoted    bool

	encoder encoderFunc
}

// byIndex sorts field by index sequence.
type byIndex []field

func (x byIndex) Len() int { return len(x) }

func (x byIndex) Swap(i, j int) { x[i], x[j] = x[j], x[i] }

func (x byIndex) Less(i, j int) bool {
	for k, xik := range x[i].index {
		if k >= len(x[j].index) {
			return false
		}
		if xik != x[j].index[k] {
			return xik < x[j].index[k]
		}
	}
	return len(x[i].index) < len(x[j].index)
}

// tagOptions is the string following a comma in a struct field's "json"
// tag, or the empty string. It does not include the leading comma.
type tagOptions string

// parseTag splits a struct field's json tag into its name and
// comma-separated options.
func parseTag(tag string) (string, tagOptions) {
	tag, opt, _ := strings.Cut(tag, ",")
	return tag, tagOptions(opt)
}

// Contains reports whether a comma-separated list of options
// contains a particular substr flag. substr must be surrounded by a
// string boundary or commas.
func (o tagOptions) Contains(optionName string) bool {
	if len(o) == 0 {
		return false
	}
	s := string(o)
	for s != "" {
		var name string
		name, s, _ = strings.Cut(s, ",")
		if name == optionName {
			return true
		}
	}
	return false
}

// typeFields returns a list of fields that LiteVectors should recognize for the given type.
// The algorithm is breadth-first search over the set of structs to include - the top struct
// and then any reachable anonymous structs.
func typeFields(t reflect.Type) structFields {
	// Anonymous fields to explore at the current level and the next.
	current := []field{}
	next := []field{{typ: t}}

	// Count of queued names for current level and the next.
	var count, nextCount map[reflect.Type]int

	// Types already visited at an earlier level.
	visited := map[reflect.Type]bool{}

	// Fields found.
	var fields []field

	for len(next) > 0 {
		current, next = next, current[:0]
		count, nextCount = nextCount, map[reflect.Type]int{}

		for _, f := range current {
			if visited[f.typ] {
				continue
			}
			visited[f.typ] = true

			// Scan f.typ for fields to include.
			for i := 0; i < f.typ.NumField(); i++ {
				sf := f.typ.Field(i)
				if sf.Anonymous {
					t := sf.Type
					if t.Kind() == reflect.Pointer {
						t = t.Elem()
					}
					if !sf.IsExported() && t.Kind() != reflect.Struct {
						// Ignore embedded fields of unexported non-struct types.
						continue
					}
					// Do not ignore embedded fields of unexported struct types
					// since they may have exported fields.
				} else if !sf.IsExported() {
					// Ignore unexported non-embedded fields.
					continue
				}
				tag := sf.Tag.Get("json")
				if tag == "-" {
					continue
				}
				name, opts := parseTag(tag)
				if !isValidTag(name) {
					name = ""
				}
				index := make([]int, len(f.index)+1)
				copy(index, f.index)
				index[len(f.index)] = i

				ft := sf.Type
				if ft.Name() == "" && ft.Kind() == reflect.Pointer {
					// Follow pointer.
					ft = ft.Elem()
				}

				// Only strings, floats, integers, and booleans can be quoted.
				quoted := false
				if opts.Contains("string") {
					switch ft.Kind() {
					case reflect.Bool,
						reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
						reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
						reflect.Float32, reflect.Float64,
						reflect.String:
						quoted = true
					}
				}

				// Record found field and index sequence.
				if name != "" || !sf.Anonymous || ft.Kind() != reflect.Struct {
					tagged := name != ""
					if name == "" {
						name = sf.Name
					}
					field := field{
						name:      name,
						tag:       tagged,
						index:     index,
						typ:       ft,
						omitEmpty: opts.Contains("omitempty"),
						quoted:    quoted,
					}
					field.nameBytes = []byte(field.name)
					//field.equalFold = foldFunc(field.nameBytes)

					field.nameNonEsc = field.name

					fields = append(fields, field)
					if count[f.typ] > 1 {
						// If there were multiple instances, add a second,
						// so that the annihilation code will see a duplicate.
						// It only cares about the distinction between 1 or 2,
						// so don't bother generating any more copies.
						fields = append(fields, fields[len(fields)-1])
					}
					continue
				}

				// Record new anonymous struct to explore in next round.
				nextCount[ft]++
				if nextCount[ft] == 1 {
					next = append(next, field{name: ft.Name(), index: index, typ: ft})
				}
			}
		}
	}

	sort.Slice(fields, func(i, j int) bool {
		x := fields
		// sort field by name, breaking ties with depth, then
		// breaking ties with "name came from json tag", then
		// breaking ties with index sequence.
		if x[i].name != x[j].name {
			return x[i].name < x[j].name
		}
		if len(x[i].index) != len(x[j].index) {
			return len(x[i].index) < len(x[j].index)
		}
		if x[i].tag != x[j].tag {
			return x[i].tag
		}
		return byIndex(x).Less(i, j)
	})

	// Delete all fields that are hidden by the Go rules for embedded fields,
	// except that fields with JSON tags are promoted.

	// The fields are sorted in primary order of name, secondary order
	// of field index length. Loop over names; for each name, delete
	// hidden fields by choosing the one dominant field that survives.
	out := fields[:0]
	for advance, i := 0, 0; i < len(fields); i += advance {
		// One iteration per name.
		// Find the sequence of fields with the name of this first field.
		fi := fields[i]
		name := fi.name
		for advance = 1; i+advance < len(fields); advance++ {
			fj := fields[i+advance]
			if fj.name != name {
				break
			}
		}
		if advance == 1 { // Only one field with this name
			out = append(out, fi)
			continue
		}
		dominant, ok := dominantField(fields[i : i+advance])
		if ok {
			out = append(out, dominant)
		}
	}

	fields = out
	sort.Sort(byIndex(fields))

	for i := range fields {
		f := &fields[i]
		f.encoder = typeEncoder(typeByIndex(t, f.index))
	}
	nameIndex := make(map[string]int, len(fields))
	for i, field := range fields {
		nameIndex[field.name] = i
	}
	return structFields{fields, nameIndex}
}

// dominantField looks through the fields, all of which are known to
// have the same name, to find the single field that dominates the
// others using Go's embedding rules, modified by the presence of
// JSON tags. If there are multiple top-level fields, the boolean
// will be false: This condition is an error in Go and we skip all
// the fields.
func dominantField(fields []field) (field, bool) {
	// The fields are sorted in increasing index-length order, then by presence of tag.
	// That means that the first field is the dominant one. We need only check
	// for error cases: two fields at top level, either both tagged or neither tagged.
	if len(fields) > 1 && len(fields[0].index) == len(fields[1].index) && fields[0].tag == fields[1].tag {
		return field{}, false
	}
	return fields[0], true
}

var fieldCache sync.Map // map[reflect.Type]structFields

// cachedTypeFields is like typeFields but uses a cache to avoid repeated work.
func cachedTypeFields(t reflect.Type) structFields {
	if f, ok := fieldCache.Load(t); ok {
		return f.(structFields)
	}
	f, _ := fieldCache.LoadOrStore(t, typeFields(t))
	return f.(structFields)
}
