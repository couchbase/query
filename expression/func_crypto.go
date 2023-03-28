//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"hash"
	"hash/crc32"
	"strconv"
	"strings"

	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
	"golang.org/x/crypto/md4"
)

type Hashbytes struct {
	FunctionBase
	table      *crc32.Table
	polynomial uint32
}

func NewHashbytes(operands ...Expression) Function {
	rv := &Hashbytes{
		*NewFunctionBase("hashbytes", operands...),
		nil,
		0,
	}

	if len(operands) == 2 && operands[1].Type() == value.OBJECT {
		if ps, ok := operands[1].Value().Field("polynomial"); ok {
			if ps.Type() == value.STRING {
				p := strings.ToLower(ps.ToString())
				rv.polynomial = parsePolynomial(p)
				rv.table = crc32.MakeTable(rv.polynomial)
			} else if ps.Type() == value.NUMBER {
				nv := ps.(value.NumberValue)
				rv.polynomial = uint32(nv.Int64())
				rv.table = crc32.MakeTable(rv.polynomial)
			}
		}
	}
	rv.expr = rv
	return rv
}

func (this *Hashbytes) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Hashbytes) Type() value.Type { return value.STRING }

var algorithms = map[string]interface{}{
	"crc32":      crc32.New,
	"md4":        md4.New,
	"md5":        md5.New,
	"sha224":     sha256.New224,
	"sha256":     sha256.New,
	"sha384":     sha512.New384,
	"sha512":     sha512.New,
	"sha512/224": sha512.New512_224,
	"sha512/256": sha512.New512_256,
}

const _DEFALT_ALGORITHM = "sha256"

func parsePolynomial(p string) uint32 {
	polynomial := uint32(crc32.IEEE)
	switch p {
	case "ieee":
		polynomial = crc32.IEEE
	case "castagnoli":
		polynomial = crc32.Castagnoli
	case "koopman":
		polynomial = crc32.Koopman
	default:
		v, err := strconv.ParseUint(p, 0, 32)
		if err == nil {
			polynomial = uint32(v)
		}
	}
	return polynomial
}

func (this *Hashbytes) Evaluate(item value.Value, context Context) (value.Value, error) {
	algo := _DEFALT_ALGORITHM
	var d []byte

	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() == value.NULL {
		return value.NULL_VALUE, nil
	}

	if arg.Type() == value.BINARY {
		d = arg.Actual().([]byte)
	} else if arg.Type() == value.STRING {
		// special case for string so as to not include the quotes found in the JSON marshalled value
		d = []byte(arg.ToString())
	} else {
		// hash the JSON representation of non-binary values
		arg.Actual() // force uwrapping of parsed values
		d, err = arg.MarshalJSON()
		if err != nil {
			return nil, err
		}
	}

	polynomial := uint32(crc32.IEEE)
	if len(this.operands) > 1 {
		options, err := this.operands[1].Evaluate(item, context)
		if err != nil {
			return nil, err
		}

		if options.Type() == value.OBJECT {
			if as, ok := options.Field("algorithm"); ok && as.Type() == value.STRING {
				a := strings.Replace(strings.ToLower(as.ToString()), "sha-", "sha", 1)
				if _, ok := algorithms[a]; ok {
					algo = a
				} else {
					return value.NULL_VALUE, nil
				}
			}
			if ps, ok := options.Field("polynomial"); ok {
				if ps.Type() == value.STRING {
					p := strings.ToLower(ps.ToString())
					polynomial = parsePolynomial(p)
				} else if ps.Type() == value.NUMBER {
					nv := ps.(value.NumberValue)
					polynomial = uint32(nv.Int64())
				}
			}
		} else if options.Type() == value.STRING {
			a := strings.Replace(strings.ToLower(options.ToString()), "sha-", "sha", 1)
			if _, ok := algorithms[a]; ok {
				algo = a
			} else {
				return value.NULL_VALUE, nil
			}
		} else if options.Type() != value.MISSING && options.Type() != value.NULL {
			return value.NULL_VALUE, nil
		}
	}

	al, ok := algorithms[algo]
	if ok {
		var r []byte
		if f, ok := al.(func() hash.Hash); ok {
			h := f()
			h.Write(d)
			r = h.Sum(nil)
		} else if f, ok := al.(func(*crc32.Table) hash.Hash32); ok {
			table := this.table
			if table == nil || this.polynomial != polynomial {
				table = crc32.MakeTable(polynomial)
			}
			h := f(table)
			h.Write(d)
			r = h.Sum(nil)
		} else {
			logging.Errorf("INTERNAL ERROR: Invalid algorithm map entry: %v - %T", algo, al)
			return value.NULL_VALUE, nil
		}
		buf := make([]byte, 0, len(r)*2)
		return value.NewValue(string(util.AppendBytesAsHex(buf, r))), nil
	}
	return value.NULL_VALUE, nil
}

func (this *Hashbytes) MinArgs() int { return 1 }

func (this *Hashbytes) MaxArgs() int { return 2 }

func (this *Hashbytes) Constructor() FunctionConstructor { return NewHashbytes }
