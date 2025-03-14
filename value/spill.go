//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package value

import (
	"encoding/binary"
	"fmt"
	"io"
	"strconv"

	json "github.com/couchbase/go_json"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/util"
)

const (
	_SPILL_TYPE_VALUE_MISSING = byte(iota + 0x80)
	_SPILL_TYPE_VALUE_NULL
	_SPILL_TYPE_VALUE_LIST
	_SPILL_TYPE_VALUE_ANNOTATED // 0x83
	_SPILL_TYPE_VALUE_ANNOTATED_SELFREF
	_SPILL_TYPE_VALUE_SCOPE
	_SPILL_TYPE_VALUE_PARSED
	_SPILL_TYPE_VALUE // 0x87
	_SPILL_TYPE_SLICE_ANNOTATED
	_SPILL_TYPE_SLICE_VALUE
	_SPILL_TYPE_SLICE_VALUES
	_SPILL_TYPE_SLICE_STRING
	_SPILL_TYPE_MAP_VALUE
	_SPILL_TYPE_MAP_VALUE_INT

	_SPILL_TYPE_MAP // 0x8e
	_SPILL_TYPE_SLICE
	_SPILL_TYPE_NIL
	_SPILL_TYPE_BOOL
	_SPILL_TYPE_BYTES
	_SPILL_TYPE_INT
	_SPILL_TYPE_INT32
	_SPILL_TYPE_UINT32
	_SPILL_TYPE_INT64
	_SPILL_TYPE_UINT64
	_SPILL_TYPE_FLOAT32
	_SPILL_TYPE_FLOAT64
	_SPILL_TYPE_STRING // 0x9a
	_SPILL_TYPE_JSON
	_SPILL_TYPE_INT16_MAP
	_SPILL_TYPE_INT16
	_SPILL_TYPE_UINT16
	_SPILL_TYPE_BYTE // 0x9f
)

const _SPILL_TYPED_NIL_INDICATOR = -1

var _SPILL_POOL = util.NewBytePool(128)

func writeSpillValue(w io.Writer, v interface{}, buf []byte) error {
	var err error
	switch v := v.(type) {
	case Value:
		err = v.WriteSpill(w, buf)
	case map[string]interface{}:
		err = writeSpillMap(w, v, buf)
	case map[string]Value:
		err = writeSpillVMap(w, v, buf)
	case map[int]Value:
		err = writeSpillIntVMap(w, v, buf)
	case map[int16]interface{}:
		err = writeSpillInt16Map(w, v, buf)
	case []interface{}:
		err = writeSpillSlice(w, v, buf)
	case []string:
		err = writeSpillSSlice(w, v, buf)
	case []AnnotatedValue:
		err = writeSpillAVSlice(w, v, buf)
	case []Value:
		err = writeSpillVSlice(_SPILL_TYPE_SLICE_VALUE, w, v, buf)
	case Values:
		err = writeSpillVSlice(_SPILL_TYPE_SLICE_VALUES, w, ([]Value)(v), buf)
	case nil:
		buf = buf[:1]
		buf[0] = _SPILL_TYPE_NIL
		_, err = w.Write(buf)
	case bool:
		buf = buf[:2]
		buf[0] = _SPILL_TYPE_BOOL
		if v {
			buf[1] = 1
		} else {
			buf[1] = 0
		}
		_, err = w.Write(buf)
	case []byte:
		l := len(v)
		if v == nil {
			l = _SPILL_TYPED_NIL_INDICATOR
		}
		err = writeSpillTypeAndLength(_SPILL_TYPE_BYTES, l, w, buf)
		if err == nil && v != nil {
			_, err = w.Write(v)
		}
	case byte:
		buf = buf[:2]
		buf[0] = _SPILL_TYPE_BYTE
		buf[1] = byte(v)
		_, err = w.Write(buf)
	case int16:
		buf = buf[:3]
		buf[0] = _SPILL_TYPE_INT16
		binary.BigEndian.PutUint16(buf[1:], uint16(v))
		_, err = w.Write(buf)
	case uint16:
		buf = buf[:3]
		buf[0] = _SPILL_TYPE_UINT16
		binary.BigEndian.PutUint16(buf[1:], uint16(v))
		_, err = w.Write(buf)
	case int32:
		buf = buf[:5]
		buf[0] = _SPILL_TYPE_INT32
		binary.BigEndian.PutUint32(buf[1:], uint32(v))
		_, err = w.Write(buf)
	case uint32:
		buf = buf[:5]
		buf[0] = _SPILL_TYPE_UINT32
		binary.BigEndian.PutUint32(buf[1:], uint32(v))
		_, err = w.Write(buf)
	case int:
		buf = buf[:9]
		buf[0] = _SPILL_TYPE_INT
		binary.BigEndian.PutUint64(buf[1:], uint64(v))
		_, err = w.Write(buf)
	case int64:
		buf = buf[:9]
		buf[0] = _SPILL_TYPE_INT64
		binary.BigEndian.PutUint64(buf[1:], uint64(v))
		_, err = w.Write(buf)
	case uint64:
		buf = buf[:9]
		buf[0] = _SPILL_TYPE_UINT64
		binary.BigEndian.PutUint64(buf[1:], uint64(v))
		_, err = w.Write(buf)
	case float32:
		buf = buf[:5]
		buf[0] = _SPILL_TYPE_FLOAT32
		buf = strconv.AppendFloat(buf, float64(v), 'e', -1, 32)
		binary.BigEndian.PutUint32(buf[1:], uint32(len(buf)-5))
		_, err = w.Write(buf)
	case float64:
		buf = buf[:5]
		buf[0] = _SPILL_TYPE_FLOAT64
		buf = strconv.AppendFloat(buf, float64(v), 'e', -1, 64)
		binary.BigEndian.PutUint32(buf[1:], uint32(len(buf)-5))
		_, err = w.Write(buf)
	case string:
		buf = buf[:5]
		buf[0] = _SPILL_TYPE_STRING
		binary.BigEndian.PutUint32(buf[1:], uint32(len(v)))
		_, err = w.Write(buf)
		if err == nil {
			_, err = w.Write([]byte(v))
		}
	default:
		logging.Debugf("writeSpillValue: writing as default type: %T", v)
		buf = buf[:5]
		buf[0] = _SPILL_TYPE_JSON
		b, err := json.Marshal(v)
		if err == nil {
			binary.BigEndian.PutUint32(buf[1:], uint32(len(b)))
			_, err = w.Write(buf)
			if err == nil {
				_, err = w.Write(b)
			}
		}
	}
	return err
}

func writeSpillTypeAndLength(typ byte, length int, w io.Writer, buf []byte) error {
	buf = buf[:5]
	buf[0] = typ
	binary.BigEndian.PutUint32(buf[1:], uint32(length))
	_, err := w.Write(buf)
	return err
}

func writeSpillMap(w io.Writer, m map[string]interface{}, buf []byte) error {
	l := len(m)
	if m == nil {
		l = _SPILL_TYPED_NIL_INDICATOR
	}
	err := writeSpillTypeAndLength(_SPILL_TYPE_MAP, l, w, buf)
	if err != nil {
		return err
	}
	for k, v := range m {
		err = writeSpillValue(w, k, buf)
		if err != nil {
			return err
		}
		err = writeSpillValue(w, v, buf)
		if err != nil {
			return err
		}
	}
	return nil
}

func writeSpillInt16Map(w io.Writer, m map[int16]interface{}, buf []byte) error {
	l := len(m)
	if m == nil {
		l = _SPILL_TYPED_NIL_INDICATOR
	}
	err := writeSpillTypeAndLength(_SPILL_TYPE_INT16_MAP, l, w, buf)
	if err != nil {
		return err
	}
	for k, v := range m {
		err = writeSpillValue(w, k, buf)
		if err != nil {
			return err
		}
		err = writeSpillValue(w, v, buf)
		if err != nil {
			return err
		}
	}
	return nil
}

func writeSpillVMap(w io.Writer, m map[string]Value, buf []byte) error {
	l := len(m)
	if m == nil {
		l = _SPILL_TYPED_NIL_INDICATOR
	}
	err := writeSpillTypeAndLength(_SPILL_TYPE_MAP_VALUE, l, w, buf)
	if err != nil {
		return err
	}
	for k, v := range m {
		err = writeSpillValue(w, k, buf)
		if err != nil {
			return err
		}
		err = writeSpillValue(w, v, buf)
		if err != nil {
			return err
		}
	}
	return nil
}

func writeSpillIntVMap(w io.Writer, m map[int]Value, buf []byte) error {
	l := len(m)
	if m == nil {
		l = _SPILL_TYPED_NIL_INDICATOR
	}
	err := writeSpillTypeAndLength(_SPILL_TYPE_MAP_VALUE_INT, l, w, buf)
	if err != nil {
		return err
	}
	for k, v := range m {
		err = writeSpillValue(w, k, buf)
		if err != nil {
			return err
		}
		err = writeSpillValue(w, v, buf)
		if err != nil {
			return err
		}
	}
	return nil
}

func writeSpillSlice(w io.Writer, s []interface{}, buf []byte) error {
	l := len(s)
	if s == nil {
		l = _SPILL_TYPED_NIL_INDICATOR
	}
	err := writeSpillTypeAndLength(_SPILL_TYPE_SLICE, l, w, buf)
	if err != nil {
		return err
	}
	for i := range s {
		err = writeSpillValue(w, s[i], buf)
		if err != nil {
			return err
		}
	}
	return nil
}

func writeSpillSSlice(w io.Writer, s []string, buf []byte) error {
	l := len(s)
	if s == nil {
		l = _SPILL_TYPED_NIL_INDICATOR
	}
	err := writeSpillTypeAndLength(_SPILL_TYPE_SLICE_STRING, l, w, buf)
	if err != nil {
		return err
	}
	for i := range s {
		err = writeSpillValue(w, s[i], buf)
		if err != nil {
			return err
		}
	}
	return nil
}

func writeSpillAVSlice(w io.Writer, s []AnnotatedValue, buf []byte) error {
	l := len(s)
	if s == nil {
		l = _SPILL_TYPED_NIL_INDICATOR
	}
	err := writeSpillTypeAndLength(_SPILL_TYPE_SLICE_ANNOTATED, l, w, buf)
	for i := range s {
		err = writeSpillValue(w, s[i], buf)
		if err != nil {
			return err
		}
	}
	return nil
}

func writeSpillVSlice(typ byte, w io.Writer, s []Value, buf []byte) error {
	l := len(s)
	if s == nil {
		l = _SPILL_TYPED_NIL_INDICATOR
	}
	err := writeSpillTypeAndLength(typ, l, w, buf)
	if err != nil {
		return err
	}
	for i := range s {
		err = writeSpillValue(w, s[i], buf)
		if err != nil {
			return err
		}
	}
	return nil
}

func readSpillValue(trackMem func(int64) error, r io.Reader, buf []byte) (interface{}, error) {
	var err error
	var v interface{}
	var n int
	var track int64
	free := false
	if buf == nil {
		buf = _SPILL_POOL.Get()
		free = true
	}
	buf = buf[:1]
	n, err = r.Read(buf)
	if err == nil && n != len(buf) {
		err = io.ErrUnexpectedEOF
	} else if err == io.ErrUnexpectedEOF {
		// compressed spill files may return ErrUnexpectedEOF here instead of just EOF, so handle this case
		err = io.EOF
	}
	if err != nil {
		if free {
			_SPILL_POOL.Put(buf)
		}
		return nil, err
	}
	switch buf[0] {
	// cases for value types
	case _SPILL_TYPE_VALUE_MISSING:
		val := NewMissingValue()
		err = val.ReadSpill(trackMem, r, buf)
		v = val
	case _SPILL_TYPE_VALUE_NULL:
		val := NewNullValue()
		err = val.ReadSpill(trackMem, r, buf)
		v = val
	case _SPILL_TYPE_VALUE_LIST:
		val := &listValue{}
		err = val.ReadSpill(trackMem, r, buf)
		v = val
	case _SPILL_TYPE_VALUE_ANNOTATED:
		val := newAnnotatedValue()
		err = val.ReadSpill(trackMem, r, buf)
		v = val
	case _SPILL_TYPE_VALUE_SCOPE:
		val := NewScopeValue(nil, nil)
		err = val.ReadSpill(trackMem, r, buf)
		v = val
	case _SPILL_TYPE_VALUE_PARSED:
		val := &parsedValue{}
		err = val.ReadSpill(trackMem, r, buf)
		v = val
	case _SPILL_TYPE_VALUE:
		var val interface{}
		val, err = readSpillValue(trackMem, r, buf)
		if err == nil {
			v = NewValue(val)
		}
	// fundamental types
	case _SPILL_TYPE_MAP:
		v, err = readSpillMap(trackMem, r, buf)
	case _SPILL_TYPE_MAP_VALUE:
		v, err = readSpillVMap(trackMem, r, buf)
	case _SPILL_TYPE_MAP_VALUE_INT:
		v, err = readSpillIntVMap(trackMem, r, buf)
	case _SPILL_TYPE_INT16_MAP:
		v, err = readSpillInt16Map(trackMem, r, buf)
	case _SPILL_TYPE_SLICE:
		v, err = readSpillSlice(trackMem, r, buf)
	case _SPILL_TYPE_SLICE_STRING:
		v, err = readSpillSSlice(trackMem, r, buf)
	case _SPILL_TYPE_SLICE_ANNOTATED:
		v, err = readSpillAVSlice(trackMem, r, buf)
	case _SPILL_TYPE_SLICE_VALUE:
		v, err = readSpillVSlice(trackMem, r, buf)
	case _SPILL_TYPE_SLICE_VALUES:
		v, err = readSpillVSlice(trackMem, r, buf)
		v = Values(v.([]Value))
	case _SPILL_TYPE_VALUE_ANNOTATED_SELFREF:
		v = (*annotatedValueSelfReference)(nil)
	case _SPILL_TYPE_NIL:
		v = nil
	case _SPILL_TYPE_BOOL:
		//buf = buf[:1]	   already this above
		n, err = r.Read(buf)
		if err == nil && n != len(buf) {
			err = io.ErrUnexpectedEOF
		}
		v = (buf[0] != 0)
		track = 1
	case _SPILL_TYPE_BYTES:
		length, err := readSpillLength(r, buf)
		if err == nil && length != _SPILL_TYPED_NIL_INDICATOR {
			// check before allocating
			if trackMem != nil {
				if err := trackMem(int64(length)); err != nil {
					return nil, err
				}
			}
			b := make([]byte, length)
			n, err = r.Read(b)
			if err == nil && n != length {
				err = io.ErrUnexpectedEOF
			}
			v = b
		}
	case _SPILL_TYPE_BYTE:
		buf = buf[:1]
		n, err = r.Read(buf)
		if err == nil && n != len(buf) {
			err = io.ErrUnexpectedEOF
		}
		v = buf[0]
		track = 1
	case _SPILL_TYPE_INT16:
		buf = buf[:2]
		n, err = r.Read(buf)
		if err == nil && n != len(buf) {
			err = io.ErrUnexpectedEOF
		}
		v = int16(binary.BigEndian.Uint16(buf))
		track = 2
	case _SPILL_TYPE_UINT16:
		buf = buf[:2]
		n, err = r.Read(buf)
		if err == nil && n != len(buf) {
			err = io.ErrUnexpectedEOF
		}
		v = uint16(binary.BigEndian.Uint16(buf))
		track = 2
	case _SPILL_TYPE_INT32:
		buf = buf[:4]
		n, err = r.Read(buf)
		if err == nil && n != len(buf) {
			err = io.ErrUnexpectedEOF
		}
		v = int32(binary.BigEndian.Uint32(buf))
		track = 4
	case _SPILL_TYPE_UINT32:
		buf = buf[:4]
		n, err = r.Read(buf)
		if err == nil && n != len(buf) {
			err = io.ErrUnexpectedEOF
		}
		v = uint32(binary.BigEndian.Uint32(buf))
		track = 4
	case _SPILL_TYPE_INT:
		buf = buf[:8]
		n, err = r.Read(buf)
		if err == nil && n != len(buf) {
			err = io.ErrUnexpectedEOF
		}
		v = int(binary.BigEndian.Uint64(buf))
		track = 8
	case _SPILL_TYPE_INT64:
		buf = buf[:8]
		n, err = r.Read(buf)
		if err == nil && n != len(buf) {
			err = io.ErrUnexpectedEOF
		}
		v = int64(binary.BigEndian.Uint64(buf))
		track = 8
	case _SPILL_TYPE_UINT64:
		buf = buf[:8]
		n, err = r.Read(buf)
		if err == nil && n != len(buf) {
			err = io.ErrUnexpectedEOF
		}
		v = uint64(binary.BigEndian.Uint64(buf))
		track = 8
	case _SPILL_TYPE_FLOAT32:
		buf = buf[:4]
		n, err = r.Read(buf)
		if err == nil {
			if n != len(buf) {
				err = io.ErrUnexpectedEOF
			} else {
				length := binary.BigEndian.Uint32(buf)
				var f float64
				if cap(buf) <= int(length) {
					buf = buf[:length]
					n, err = r.Read(buf)
					if err == nil {
						if n != int(length) {
							err = io.ErrUnexpectedEOF
						} else {
							f, err = strconv.ParseFloat(string(buf), 32)
						}
					}
				} else {
					b := make([]byte, length)
					n, err = r.Read(b)
					if err == nil {
						if n != int(length) {
							err = io.ErrUnexpectedEOF
						} else {
							f, err = strconv.ParseFloat(string(b), 32)
						}
					}
				}
				if err == nil {
					v = float32(f)
				}
			}
			track = 4
		}
	case _SPILL_TYPE_FLOAT64:
		buf = buf[:4]
		n, err = r.Read(buf)
		if err == nil {
			if n != len(buf) {
				err = io.ErrUnexpectedEOF
			} else {
				length := binary.BigEndian.Uint32(buf)
				if cap(buf) <= int(length) {
					buf = buf[:length]
					n, err = r.Read(buf)
					if err == nil {
						if n != int(length) {
							err = io.ErrUnexpectedEOF
						} else {
							v, err = strconv.ParseFloat(string(buf), 64)
						}
					}
				} else {
					b := make([]byte, length)
					n, err = r.Read(b)
					if err == nil {
						if n != int(length) {
							err = io.ErrUnexpectedEOF
						} else {
							v, err = strconv.ParseFloat(string(b), 64)
						}
					}
				}
			}
			track = 8
		}
	case _SPILL_TYPE_STRING:
		buf = buf[:4]
		n, err = r.Read(buf)
		if err == nil {
			if n != len(buf) {
				err = io.ErrUnexpectedEOF
			} else {
				length := uint32(binary.BigEndian.Uint32(buf))
				// check before allocating
				if trackMem != nil {
					if err := trackMem(int64(length + _INTERFACE_SIZE)); err != nil {
						return nil, err
					}
				}
				sb := make([]byte, length)
				_, err = r.Read(sb)
				if err == nil {
					v = string(sb)
				}
			}
		}
	case _SPILL_TYPE_JSON:
		buf = buf[:4]
		n, err = r.Read(buf)
		if err == nil {
			if n != len(buf) {
				err = io.ErrUnexpectedEOF
			} else {
				length := uint32(binary.BigEndian.Uint32(buf))
				jb := make([]byte, length)
				n, err = r.Read(jb)
				if err == nil {
					if n != int(length) {
						err = io.ErrUnexpectedEOF
					} else {
						err = json.Unmarshal(jb, &v)
					}
				}
				track = int64(AnySize(v))
			}
		}
	default:
		panic(fmt.Sprintf("Unknown spill file element type: %v", buf[0]))
	}
	if free {
		_SPILL_POOL.Put(buf)
	}
	if err == nil && trackMem != nil && track != 0 {
		if err := trackMem(track); err != nil {
			return nil, err
		}
	}
	return v, err
}

func readSpillLength(r io.Reader, buf []byte) (int, error) {
	buf = buf[:4]
	n, err := r.Read(buf)
	if err == nil && n != len(buf) {
		err = io.ErrUnexpectedEOF
	}
	if err != nil {
		return 0, err
	}
	length := int(int32(binary.BigEndian.Uint32(buf)))
	return length, err
}

func readSpillMap(trackMem func(int64) error, r io.Reader, buf []byte) (map[string]interface{}, error) {
	length, err := readSpillLength(r, buf)
	if err != nil {
		return nil, err
	}
	if length == _SPILL_TYPED_NIL_INDICATOR {
		return (map[string]interface{})(nil), nil
	}
	if trackMem != nil {
		if err := trackMem(int64(mapBaseSize(length))); err != nil {
			return nil, err
		}
	}
	m := make(map[string]interface{}, length)
	var k, v interface{}
	for i := 0; i < length; i++ {
		k, err = readSpillValue(trackMem, r, buf)
		if err != nil {
			return nil, err
		}
		v, err = readSpillValue(trackMem, r, buf)
		if err != nil {
			return nil, err
		}
		m[k.(string)] = v
	}
	return m, nil
}

func readSpillInt16Map(trackMem func(int64) error, r io.Reader, buf []byte) (map[int16]interface{}, error) {
	length, err := readSpillLength(r, buf)
	if err != nil {
		return nil, err
	}
	if length == _SPILL_TYPED_NIL_INDICATOR {
		return (map[int16]interface{})(nil), nil
	}
	if trackMem != nil {
		if err := trackMem(int64(mapBaseSize(length))); err != nil {
			return nil, err
		}
	}
	m := make(map[int16]interface{}, length)
	var k, v interface{}
	for i := 0; i < length; i++ {
		k, err = readSpillValue(trackMem, r, buf)
		if err != nil {
			return nil, err
		}
		v, err = readSpillValue(trackMem, r, buf)
		if err != nil {
			return nil, err
		}
		m[k.(int16)] = v
	}
	return m, nil
}

func readSpillVMap(trackMem func(int64) error, r io.Reader, buf []byte) (map[string]Value, error) {
	length, err := readSpillLength(r, buf)
	if err != nil {
		return nil, err
	}
	if length == _SPILL_TYPED_NIL_INDICATOR {
		return (map[string]Value)(nil), nil
	}
	if trackMem != nil {
		if err := trackMem(int64(mapBaseSize(length))); err != nil {
			return nil, err
		}
	}
	m := make(map[string]Value)
	var k, v interface{}
	for i := 0; i < length; i++ {
		k, err = readSpillValue(trackMem, r, buf)
		if err != nil {
			return nil, err
		}
		v, err = readSpillValue(trackMem, r, buf)
		if err != nil {
			return nil, err
		}
		m[k.(string)] = v.(Value)
	}
	return m, nil
}

func readSpillIntVMap(trackMem func(int64) error, r io.Reader, buf []byte) (map[int]Value, error) {
	length, err := readSpillLength(r, buf)
	if err != nil {
		return nil, err
	}
	if length == _SPILL_TYPED_NIL_INDICATOR {
		return (map[int]Value)(nil), nil
	}
	if trackMem != nil {
		if err := trackMem(int64(mapBaseSize(length))); err != nil {
			return nil, err
		}
	}
	m := make(map[int]Value)
	var k, v interface{}
	for i := 0; i < length; i++ {
		k, err = readSpillValue(trackMem, r, buf)
		if err != nil {
			return nil, err
		}
		v, err = readSpillValue(trackMem, r, buf)
		if err != nil {
			return nil, err
		}
		m[k.(int)] = v.(Value)
	}
	return m, nil
}

func readSpillSlice(trackMem func(int64) error, r io.Reader, buf []byte) ([]interface{}, error) {
	length, err := readSpillLength(r, buf)
	if err != nil {
		return nil, err
	}
	if length == _SPILL_TYPED_NIL_INDICATOR {
		return ([]interface{})(nil), nil
	}
	if trackMem != nil {
		if err := trackMem(int64(_INTERFACE_SIZE * length)); err != nil {
			return nil, err
		}
	}
	s := make([]interface{}, length)
	for i := 0; i < length; i++ {
		s[i], err = readSpillValue(trackMem, r, buf)
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

func readSpillSSlice(trackMem func(int64) error, r io.Reader, buf []byte) ([]string, error) {
	length, err := readSpillLength(r, buf)
	if err != nil {
		return nil, err
	}
	if length == _SPILL_TYPED_NIL_INDICATOR {
		return ([]string)(nil), nil
	}
	if trackMem != nil {
		if err := trackMem(int64(_INTERFACE_SIZE * length)); err != nil {
			return nil, err
		}
	}
	s := make([]string, length)
	var v interface{}
	for i := 0; i < length; i++ {
		v, err = readSpillValue(trackMem, r, buf)
		if err != nil {
			return nil, err
		}
		s[i] = v.(string)
	}
	return s, nil
}

func readSpillVSlice(trackMem func(int64) error, r io.Reader, buf []byte) ([]Value, error) {
	length, err := readSpillLength(r, buf)
	if err != nil {
		return nil, err
	}
	if length == _SPILL_TYPED_NIL_INDICATOR {
		return ([]Value)(nil), nil
	}
	if trackMem != nil {
		if err := trackMem(int64(_INTERFACE_SIZE * length)); err != nil {
			return nil, err
		}
	}
	s := make([]Value, length)
	var v interface{}
	for i := 0; i < length; i++ {
		v, err = readSpillValue(trackMem, r, buf)
		if err != nil {
			return nil, err
		}
		s[i] = v.(Value)
	}
	return s, nil
}

func readSpillAVSlice(trackMem func(int64) error, r io.Reader, buf []byte) ([]AnnotatedValue, error) {
	length, err := readSpillLength(r, buf)
	if err != nil {
		return nil, err
	}
	if length == _SPILL_TYPED_NIL_INDICATOR {
		return ([]AnnotatedValue)(nil), nil
	}
	if trackMem != nil {
		if err := trackMem(int64(_INTERFACE_SIZE * length)); err != nil {
			return nil, err
		}
	}
	s := make([]AnnotatedValue, length)
	var v interface{}
	for i := 0; i < length; i++ {
		v, err = readSpillValue(trackMem, r, buf)
		if err != nil {
			return nil, err
		}
		s[i] = v.(AnnotatedValue)
	}
	return s, nil
}
