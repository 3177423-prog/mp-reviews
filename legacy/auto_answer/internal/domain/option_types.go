package domain

import (
	"time"

	"github.com/go-faster/errors"
	"github.com/go-faster/jx"
)

// ********************************************************** OptBool ************************************************************

type OptBool struct {
	Value bool
	Set   bool
}

func (o OptBool) IsSet() bool { return o.Set }

func (o *OptBool) Reset() {
	var v bool
	o.Value = v
	o.Set = false
}

func (o *OptBool) SetTo(v bool) {
	o.Set = true
	o.Value = v
}

func (o OptBool) Get() (v bool, ok bool) {
	if !o.Set {
		return v, false
	}
	return o.Value, true
}

func (o OptBool) Or(d bool) bool {
	if v, ok := o.Get(); ok {
		return v
	}
	return d
}

func (o OptBool) Encode(e *jx.Encoder) {
	if !o.Set {
		return
	}
	e.Bool(bool(o.Value))
}

func (o *OptBool) Decode(d *jx.Decoder) error {
	if o == nil {
		return errors.New("недействительно: невозможно декодировать OptBool в nil")
	}
	o.Set = true
	v, err := d.Bool()
	if err != nil {
		return err
	}
	o.Value = bool(v)
	return nil
}

func (o OptBool) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	o.Encode(&e)
	return e.Bytes(), nil
}

func (o *OptBool) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return o.Decode(d)
}

// ********************************************************** OptDate ************************************************************

type OptDate struct {
	Value time.Time
	Set   bool
}

func (o OptDate) IsSet() bool { return o.Set }

func (o *OptDate) Reset() {
	var v time.Time
	o.Value = v
	o.Set = false
}

func (o *OptDate) SetTo(v time.Time) {
	o.Set = true
	o.Value = v
}

func (o OptDate) Get() (v time.Time, ok bool) {
	if !o.Set {
		return v, false
	}
	return o.Value, true
}

func (o OptDate) Or(d time.Time) time.Time {
	if v, ok := o.Get(); ok {
		return v
	}
	return d
}

func (o OptDate) Encode(e *jx.Encoder, format func(*jx.Encoder, time.Time)) {
	if !o.Set {
		return
	}
	format(e, o.Value)
}

func (o *OptDate) Decode(d *jx.Decoder, format func(*jx.Decoder) (time.Time, error)) error {
	if o == nil {
		return errors.New("недействительно: невозможно декодировать OptDate в nil")
	}
	o.Set = true
	v, err := format(d)
	if err != nil {
		return err
	}
	o.Value = v
	return nil
}

func (o OptDate) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	o.Encode(&e, EncodeDate)
	return e.Bytes(), nil
}

func (o *OptDate) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return o.Decode(d, DecodeDate)
}

// ********************************************************** OptDateTime ************************************************************

type OptDateTime struct {
	Value time.Time
	Set   bool
}

func (o OptDateTime) IsSet() bool { return o.Set }

func (o *OptDateTime) Reset() {
	var v time.Time
	o.Value = v
	o.Set = false
}

func (o *OptDateTime) SetTo(v time.Time) {
	o.Set = true
	o.Value = v
}

func (o OptDateTime) Get() (v time.Time, ok bool) {
	if !o.Set {
		return v, false
	}
	return o.Value, true
}

func (o OptDateTime) Or(d time.Time) time.Time {
	if v, ok := o.Get(); ok {
		return v
	}
	return d
}

func (o OptDateTime) Encode(e *jx.Encoder, format func(*jx.Encoder, time.Time)) {
	if !o.Set {
		return
	}
	format(e, o.Value)
}

func (o *OptDateTime) Decode(d *jx.Decoder, format func(*jx.Decoder) (time.Time, error)) error {
	if o == nil {
		return errors.New("недействительно: невозможно декодировать OptDateTime в nil")
	}
	o.Set = true
	v, err := format(d)
	if err != nil {
		return err
	}
	o.Value = v
	return nil
}

func (o OptDateTime) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	o.Encode(&e, EncodeDateTime)
	return e.Bytes(), nil
}

func (o *OptDateTime) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return o.Decode(d, DecodeDateTime)
}

// ********************************************************** OptFloat32 ************************************************************

type OptFloat32 struct {
	Value float32
	Set   bool
}

func NewOptFloat32(v float32) OptFloat32 {
	return OptFloat32{
		Value: v,
		Set:   true,
	}
}

func (o OptFloat32) IsSet() bool { return o.Set }

func (o *OptFloat32) Reset() {
	var v float32
	o.Value = v
	o.Set = false
}

func (o *OptFloat32) SetTo(v float32) {
	o.Set = true
	o.Value = v
}

func (o OptFloat32) Get() (v float32, ok bool) {
	if !o.Set {
		return v, false
	}
	return o.Value, true
}

func (o OptFloat32) Or(d float32) float32 {
	if v, ok := o.Get(); ok {
		return v
	}
	return d
}

func (o OptFloat32) Encode(e *jx.Encoder) {
	if !o.Set {
		return
	}
	e.Float32(float32(o.Value))
}

func (o *OptFloat32) Decode(d *jx.Decoder) error {
	if o == nil {
		return errors.New("недействительно: невозможно декодировать OptFloat32 в nil")
	}
	o.Set = true
	v, err := d.Float32()
	if err != nil {
		return err
	}
	o.Value = float32(v)
	return nil
}

func (o OptFloat32) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	o.Encode(&e)
	return e.Bytes(), nil
}

func (o *OptFloat32) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return o.Decode(d)
}

// ********************************************************** OptFloat64 ************************************************************

type OptFloat64 struct {
	Value float64
	Set   bool
}

func (o OptFloat64) IsSet() bool { return o.Set }

func (o *OptFloat64) Reset() {
	var v float64
	o.Value = v
	o.Set = false
}

func (o *OptFloat64) SetTo(v float64) {
	o.Set = true
	o.Value = v
}

func (o OptFloat64) Get() (v float64, ok bool) {
	if !o.Set {
		return v, false
	}
	return o.Value, true
}

func (o OptFloat64) Or(d float64) float64 {
	if v, ok := o.Get(); ok {
		return v
	}
	return d
}

func (o OptFloat64) Encode(e *jx.Encoder) {
	if !o.Set {
		return
	}
	e.Float64(float64(o.Value))
}

func (o *OptFloat64) Decode(d *jx.Decoder) error {
	if o == nil {
		return errors.New("недействительно: невозможно декодировать OptFloat64 в nil")
	}
	o.Set = true
	v, err := d.Float64()
	if err != nil {
		return err
	}
	o.Value = float64(v)
	return nil
}

func (o OptFloat64) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	o.Encode(&e)
	return e.Bytes(), nil
}

func (o *OptFloat64) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return o.Decode(d)
}

// ********************************************************** OptInt ************************************************************

type OptInt struct {
	Value int
	Set   bool
}

func (o OptInt) IsSet() bool { return o.Set }

func (o *OptInt) Reset() {
	var v int
	o.Value = v
	o.Set = false
}

func (o *OptInt) SetTo(v int) {
	o.Set = true
	o.Value = v
}

func (o OptInt) Get() (v int, ok bool) {
	if !o.Set {
		return v, false
	}
	return o.Value, true
}

func (o OptInt) Or(d int) int {
	if v, ok := o.Get(); ok {
		return v
	}
	return d
}

func (o OptInt) Encode(e *jx.Encoder) {
	if !o.Set {
		return
	}
	e.Int(int(o.Value))
}

func (o *OptInt) Decode(d *jx.Decoder) error {
	if o == nil {
		return errors.New("недействительно: невозможно декодировать OptInt в nil")
	}
	o.Set = true
	v, err := d.Int()
	if err != nil {
		return err
	}
	o.Value = int(v)
	return nil
}

func (o OptInt) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	o.Encode(&e)
	return e.Bytes(), nil
}

func (o *OptInt) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return o.Decode(d)
}

// ********************************************************** OptInt8 ************************************************************

type OptInt8 struct {
	Value int8
	Set   bool
}

func (o OptInt8) IsSet() bool { return o.Set }

func (o *OptInt8) Reset() {
	var v int8
	o.Value = v
	o.Set = false
}

func (o *OptInt8) SetTo(v int8) {
	o.Set = true
	o.Value = v
}

func (o OptInt8) Get() (v int8, ok bool) {
	if !o.Set {
		return v, false
	}
	return o.Value, true
}

func (o OptInt8) Or(d int8) int8 {
	if v, ok := o.Get(); ok {
		return v
	}
	return d
}

func (o OptInt8) Encode(e *jx.Encoder) {
	if !o.Set {
		return
	}
	e.Int8(int8(o.Value))
}

func (o *OptInt8) Decode(d *jx.Decoder) error {
	if o == nil {
		return errors.New("недействительно: невозможно декодировать OptInt8 в nil")
	}
	o.Set = true
	v, err := d.Int8()
	if err != nil {
		return err
	}
	o.Value = int8(v)
	return nil
}

func (o OptInt8) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	o.Encode(&e)
	return e.Bytes(), nil
}

func (o *OptInt8) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return o.Decode(d)
}

// ********************************************************** OptInt16 ************************************************************

type OptInt16 struct {
	Value int16
	Set   bool
}

func (o OptInt16) IsSet() bool { return o.Set }

func (o *OptInt16) Reset() {
	var v int16
	o.Value = v
	o.Set = false
}

func (o *OptInt16) SetTo(v int16) {
	o.Set = true
	o.Value = v
}

func (o OptInt16) Get() (v int16, ok bool) {
	if !o.Set {
		return v, false
	}
	return o.Value, true
}

func (o OptInt16) Or(d int16) int16 {
	if v, ok := o.Get(); ok {
		return v
	}
	return d
}

func (o OptInt16) Encode(e *jx.Encoder) {
	if !o.Set {
		return
	}
	e.Int16(int16(o.Value))
}

func (o *OptInt16) Decode(d *jx.Decoder) error {
	if o == nil {
		return errors.New("недействительно: невозможно декодировать OptInt16 в nil")
	}
	o.Set = true
	v, err := d.Int16()
	if err != nil {
		return err
	}
	o.Value = int16(v)
	return nil
}

func (o OptInt16) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	o.Encode(&e)
	return e.Bytes(), nil
}

func (o *OptInt16) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return o.Decode(d)
}

// ********************************************************** OptInt32 ************************************************************

type OptInt32 struct {
	Value int32
	Set   bool
}

func (o OptInt32) IsSet() bool { return o.Set }

func (o *OptInt32) Reset() {
	var v int32
	o.Value = v
	o.Set = false
}

func (o *OptInt32) SetTo(v int32) {
	o.Set = true
	o.Value = v
}

func (o OptInt32) Get() (v int32, ok bool) {
	if !o.Set {
		return v, false
	}
	return o.Value, true
}

func (o OptInt32) Or(d int32) int32 {
	if v, ok := o.Get(); ok {
		return v
	}
	return d
}

func (o OptInt32) Encode(e *jx.Encoder) {
	if !o.Set {
		return
	}
	e.Int32(int32(o.Value))
}

func (o *OptInt32) Decode(d *jx.Decoder) error {
	if o == nil {
		return errors.New("недействительно: невозможно декодировать OptInt32 в nil")
	}
	o.Set = true
	v, err := d.Int32()
	if err != nil {
		return err
	}
	o.Value = int32(v)
	return nil
}

func (o OptInt32) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	o.Encode(&e)
	return e.Bytes(), nil
}

func (o *OptInt32) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return o.Decode(d)
}

// ********************************************************** OptInt64 ************************************************************

type OptInt64 struct {
	Value int64
	Set   bool
}

func (o OptInt64) IsSet() bool { return o.Set }

func (o *OptInt64) Reset() {
	var v int64
	o.Value = v
	o.Set = false
}

func (o *OptInt64) SetTo(v int64) {
	o.Set = true
	o.Value = v
}

func (o OptInt64) Get() (v int64, ok bool) {
	if !o.Set {
		return v, false
	}
	return o.Value, true
}

func (o OptInt64) Or(d int64) int64 {
	if v, ok := o.Get(); ok {
		return v
	}
	return d
}

func (o OptInt64) Encode(e *jx.Encoder) {
	if !o.Set {
		return
	}
	e.Int64(int64(o.Value))
}

func (o *OptInt64) Decode(d *jx.Decoder) error {
	if o == nil {
		return errors.New("недействительно: невозможно декодировать OptInt64 в nil")
	}
	o.Set = true
	v, err := d.Int64()
	if err != nil {
		return err
	}
	o.Value = int64(v)
	return nil
}

func (o OptInt64) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	o.Encode(&e)
	return e.Bytes(), nil
}

func (o *OptInt64) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return o.Decode(d)
}

// ********************************************************** OptNilString ************************************************************

type OptNilString struct {
	Value string
	Set   bool
	Null  bool
}

func (o OptNilString) IsSet() bool { return o.Set }

func (o *OptNilString) Reset() {
	var v string
	o.Value = v
	o.Set = false
	o.Null = false
}

func (o *OptNilString) SetTo(v string) {
	o.Set = true
	o.Null = false
	o.Value = v
}

func (o OptNilString) IsNull() bool { return o.Null }

func (o *OptNilString) SetToNull() {
	o.Set = true
	o.Null = true
	var v string
	o.Value = v
}

func (o OptNilString) Get() (v string, ok bool) {
	if o.Null {
		return v, false
	}
	if !o.Set {
		return v, false
	}
	return o.Value, true
}

func (o OptNilString) Or(d string) string {
	if v, ok := o.Get(); ok {
		return v
	}
	return d
}

func (o OptNilString) Encode(e *jx.Encoder) {
	if !o.Set {
		return
	}
	if o.Null {
		e.Null()
		return
	}
	e.Str(string(o.Value))
}

func (o *OptNilString) Decode(d *jx.Decoder) error {
	if o == nil {
		return errors.New("недействительно: невозможно декодировать OptNilString в nil")
	}
	if d.Next() == jx.Null {
		if err := d.Null(); err != nil {
			return err
		}

		var v string
		o.Value = v
		o.Set = true
		o.Null = true
		return nil
	}
	o.Set = true
	o.Null = false
	v, err := d.Str()
	if err != nil {
		return err
	}
	o.Value = string(v)
	return nil
}

func (o OptNilString) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	o.Encode(&e)
	return e.Bytes(), nil
}

func (o *OptNilString) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return o.Decode(d)
}

// ********************************************************** OptString ************************************************************

type OptString struct {
	Value string
	Set   bool
}

func (o OptString) IsSet() bool { return o.Set }

func (o *OptString) Reset() {
	var v string
	o.Value = v
	o.Set = false
}

func (o *OptString) SetTo(v string) {
	o.Set = true
	o.Value = v
}

func (o OptString) Get() (v string, ok bool) {
	if !o.Set {
		return v, false
	}
	return o.Value, true
}

func (o OptString) Or(d string) string {
	if v, ok := o.Get(); ok {
		return v
	}
	return d
}

func (o OptString) Encode(e *jx.Encoder) {
	if !o.Set {
		return
	}
	e.Str(string(o.Value))
}

func (o *OptString) Decode(d *jx.Decoder) error {
	if o == nil {
		return errors.New("недействительно: невозможно декодировать OptString в nil")
	}
	o.Set = true
	v, err := d.Str()
	if err != nil {
		return err
	}
	o.Value = string(v)
	return nil
}

func (o OptString) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	o.Encode(&e)
	return e.Bytes(), nil
}

func (o *OptString) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return o.Decode(d)
}

// ********************************************************** OptUint ************************************************************

type OptUint struct {
	Value uint
	Set   bool
}

func (o OptUint) IsSet() bool { return o.Set }

func (o *OptUint) Reset() {
	var v uint
	o.Value = v
	o.Set = false
}

func (o *OptUint) SetTo(v uint) {
	o.Set = true
	o.Value = v
}

func (o OptUint) Get() (v uint, ok bool) {
	if !o.Set {
		return v, false
	}
	return o.Value, true
}

func (o OptUint) Or(d uint) uint {
	if v, ok := o.Get(); ok {
		return v
	}
	return d
}

func (o OptUint) Encode(e *jx.Encoder) {
	if !o.Set {
		return
	}
	e.UInt(uint(o.Value))
}

func (o *OptUint) Decode(d *jx.Decoder) error {
	if o == nil {
		return errors.New("недействительно: невозможно декодировать OptUint в nil")
	}
	o.Set = true
	v, err := d.UInt()
	if err != nil {
		return err
	}
	o.Value = uint(v)
	return nil
}

func (o OptUint) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	o.Encode(&e)
	return e.Bytes(), nil
}

func (o *OptUint) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return o.Decode(d)
}

// ********************************************************** OptUInt8 ************************************************************

type OptUInt8 struct {
	Value uint8
	Set   bool
}

func (o OptUInt8) IsSet() bool { return o.Set }

func (o *OptUInt8) Reset() {
	var v uint8
	o.Value = v
	o.Set = false
}

func (o *OptUInt8) SetTo(v uint8) {
	o.Set = true
	o.Value = v
}

func (o OptUInt8) Get() (v uint8, ok bool) {
	if !o.Set {
		return v, false
	}
	return o.Value, true
}

func (o OptUInt8) Or(d uint8) uint8 {
	if v, ok := o.Get(); ok {
		return v
	}
	return d
}

func (o OptUInt8) Encode(e *jx.Encoder) {
	if !o.Set {
		return
	}
	e.UInt8(uint8(o.Value))
}

func (o *OptUInt8) Decode(d *jx.Decoder) error {
	if o == nil {
		return errors.New("недействительно: невозможно декодировать OptUInt8 в nil")
	}
	o.Set = true
	v, err := d.Int8()
	if err != nil {
		return err
	}
	o.Value = uint8(v)
	return nil
}

func (o OptUInt8) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	o.Encode(&e)
	return e.Bytes(), nil
}

func (o *OptUInt8) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return o.Decode(d)
}

// ********************************************************** OptNilStringArray ************************************************************

type OptNilStringArray struct {
	Value []string
	Set   bool
	Null  bool
}

func (o OptNilStringArray) IsSet() bool { return o.Set }

func (o *OptNilStringArray) Reset() {
	var v []string
	o.Value = v
	o.Set = false
	o.Null = false
}

func (o OptNilStringArray) IsNull() bool { return o.Null }

func (o OptNilStringArray) Or(d []string) []string {
	if v, ok := o.Get(); ok {
		return v
	}
	return d
}

func (o OptNilStringArray) Get() (v []string, ok bool) {
	if !o.Set {
		return v, false
	}
	return o.Value, true
}

func (o OptNilStringArray) Encode(e *jx.Encoder) {
	if !o.Set {
		return
	}
	if o.Null {
		e.Null()
		return
	}
	e.ArrStart()
	for _, elem := range o.Value {
		e.Str(elem)
	}
	e.ArrEnd()
}

func (o *OptNilStringArray) Decode(d *jx.Decoder) error {
	if o == nil {
		return errors.New("invalid: unable to decode OptNilStringArray to nil")
	}
	if d.Next() == jx.Null {
		if err := d.Null(); err != nil {
			return err
		}

		var v []string
		o.Value = v
		o.Set = true
		o.Null = true
		return nil
	}
	o.Set = true
	o.Null = false
	o.Value = make([]string, 0)
	if err := d.Arr(func(d *jx.Decoder) error {
		var elem string
		v, err := d.Str()
		elem = string(v)
		if err != nil {
			return err
		}
		o.Value = append(o.Value, elem)
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (s OptNilStringArray) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	s.Encode(&e)
	return e.Bytes(), nil
}

func (s *OptNilStringArray) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return s.Decode(d)
}

// ********************************************************** OptNilInt32Array ************************************************************

type OptNilInt32Array struct {
	Value []int32
	Set   bool
	Null  bool
}

func (o OptNilInt32Array) IsSet() bool { return o.Set }

func (o *OptNilInt32Array) Reset() {
	var v []int32
	o.Value = v
	o.Set = false
	o.Null = false
}

func (o OptNilInt32Array) IsNull() bool { return o.Null }

func (o OptNilInt32Array) Or(d []int32) []int32 {
	if v, ok := o.Get(); ok {
		return v
	}
	return d
}

func (o OptNilInt32Array) Get() (v []int32, ok bool) {
	if !o.Set {
		return v, false
	}
	return o.Value, true
}

func (o OptNilInt32Array) Encode(e *jx.Encoder) {
	if !o.Set {
		return
	}
	if o.Null {
		e.Null()
		return
	}
	e.ArrStart()
	for _, elem := range o.Value {
		e.Int32(elem)
	}
	e.ArrEnd()
}

func (o *OptNilInt32Array) Decode(d *jx.Decoder) error {
	if o == nil {
		return errors.New("invalid: unable to decode OptNilInt32Array to nil")
	}
	if d.Next() == jx.Null {
		if err := d.Null(); err != nil {
			return err
		}

		var v []int32
		o.Value = v
		o.Set = true
		o.Null = true
		return nil
	}
	o.Set = true
	o.Null = false
	o.Value = make([]int32, 0)
	if err := d.Arr(func(d *jx.Decoder) error {
		var elem int32
		v, err := d.Int32()
		elem = int32(v)
		if err != nil {
			return err
		}
		o.Value = append(o.Value, elem)
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (s OptNilInt32Array) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	s.Encode(&e)
	return e.Bytes(), nil
}

func (s *OptNilInt32Array) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return s.Decode(d)
}

// ********************************************************** OptNilInt64Array ************************************************************

type OptNilInt64Array struct {
	Value []int64
	Set   bool
	Null  bool
}

func (o OptNilInt64Array) IsSet() bool { return o.Set }

func (o *OptNilInt64Array) Reset() {
	var v []int64
	o.Value = v
	o.Set = false
	o.Null = false
}

func (o OptNilInt64Array) IsNull() bool { return o.Null }

func (o OptNilInt64Array) Or(d []int64) []int64 {
	if v, ok := o.Get(); ok {
		return v
	}
	return d
}

func (o OptNilInt64Array) Get() (v []int64, ok bool) {
	if !o.Set {
		return v, false
	}
	return o.Value, true
}

func (o OptNilInt64Array) Encode(e *jx.Encoder) {
	if !o.Set {
		return
	}
	if o.Null {
		e.Null()
		return
	}
	e.ArrStart()
	for _, elem := range o.Value {
		e.Int64(elem)
	}
	e.ArrEnd()
}

func (o *OptNilInt64Array) Decode(d *jx.Decoder) error {
	if o == nil {
		return errors.New("invalid: unable to decode OptNilInt64Array to nil")
	}
	if d.Next() == jx.Null {
		if err := d.Null(); err != nil {
			return err
		}

		var v []int64
		o.Value = v
		o.Set = true
		o.Null = true
		return nil
	}
	o.Set = true
	o.Null = false
	o.Value = make([]int64, 0)
	if err := d.Arr(func(d *jx.Decoder) error {
		var elem int64
		v, err := d.Int64()
		elem = int64(v)
		if err != nil {
			return err
		}
		o.Value = append(o.Value, elem)
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (s OptNilInt64Array) MarshalJSON() ([]byte, error) {
	e := jx.Encoder{}
	s.Encode(&e)
	return e.Bytes(), nil
}

func (s *OptNilInt64Array) UnmarshalJSON(data []byte) error {
	d := jx.DecodeBytes(data)
	return s.Decode(d)
}
