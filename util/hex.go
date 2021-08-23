//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package util

import ()

func AppendBytesAsHex(d, s []byte) []byte {
	for _, b := range s {
		b1 := hexNum(b >> 4)
		b2 := hexNum(b & 0xf)
		d = append(d, b1, b2)
	}
	return d
}

func hexNum(b byte) byte {
	if b < 10 {
		b = b + '0'
	} else {
		b = b - 10 + 'a'
	}
	return b
}

func AppendBytesFromHex(res []byte, src []byte) []byte {
	if len(src)&1 != 0 {
		return nil
	}
	for i := 0; i < len(src); {
		h := hexVal(src[i])
		if h == 255 {
			return nil
		}
		i++
		l := hexVal(src[i])
		if l == 255 {
			return nil
		}
		i++
		res = append(res, (h<<4)|l)
	}
	return res
}

func hexVal(b byte) byte {
	if b >= '0' && b <= '9' {
		return b & 0x0f
	} else if b >= 'A' && b <= 'F' {
		return b - 'A' + 10
	} else if b >= 'a' && b <= 'f' {
		return b - 'a' + 10
	}
	return 255
}
