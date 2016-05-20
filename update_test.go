// Copyright (c) 2010-2012 - Gustavo Niemeyer <gustavo@niemeyer.net>, Canonical Ltd
//
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation
//    and/or other materials provided with the distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package mgoutil_test

import (
	"encoding/binary"
	"errors"

	gc "gopkg.in/check.v1"
	"gopkg.in/mgo.v2/bson"
)

type S struct{}

var _ = gc.Suite(&S{})

var asUpdateTests = []struct {
	description string
	obj         interface{}
	expect      bson.Update
	expectError string
}{{
	obj: struct{ X int }{},
	expect: bson.Update{
		Set:   bson.M{"x": 0},
		Unset: bson.M{},
	},
}, {
	obj: struct {
		X int `bson:"y"`
	}{},
	expect: bson.Update{
		Set:   bson.M{"y": 0},
		Unset: bson.M{},
	},
}, {
	obj: struct {
		X int `bson:",omitempty"`
		Y string
	}{
		Y: "hello",
	},
	expect: bson.Update{
		Set:   bson.M{"y": "hello"},
		Unset: bson.M{"x": nil},
	},
}, {
	obj: map[string]interface{}{
		"A": "b",
		"C": 213,
	},
	expect: bson.Update{
		Set:   bson.M{"A": "b", "C": 213},
		Unset: bson.M{},
	},
}, {
	obj: &typeWithGetter{
		result: &typeWithGetter{
			result: struct{ A int }{1},
		},
	},
	expect: bson.Update{
		Set:   bson.M{"a": 1},
		Unset: bson.M{},
	},
}, {
	obj: &bson.Raw{0x03, []byte(wrapInDoc("\x0Aa\x00\x0Ac\x00\x0Ab\x00\x08d\x00\x01"))},
	expect: bson.Update{
		Set: bson.M{
			"a": bson.Raw{0x0a, []byte{}},
			"c": bson.Raw{0x0a, []byte{}},
			"b": bson.Raw{0x0a, []byte{}},
			"d": bson.Raw{0x08, []byte("\x01")},
		},
		Unset: bson.M{},
	},
}, {
	obj:         34,
	expectError: `cannot marshal: Can't marshal int as a BSON document`,
}, {
	obj: &typeWithGetter{
		err: errors.New("some error"),
	},
	expectError: `GetBSON failed: some error`,
}, {
	obj: &inlineInt{struct{ A, B int }{1, 2}},
	expect: bson.Update{
		Set: bson.M{
			"a": 1,
			"b": 2,
		},
		Unset: bson.M{},
	},
}, {
	obj: &inlineMap{A: 1, M: map[string]interface{}{"b": 2}},
	expect: bson.Update{
		Set: bson.M{
			"a": 1,
			"b": 2,
		},
		Unset: bson.M{},
	},
}, {
	obj:         &structWithDupKeys{},
	expectError: `Duplicated key 'name' in struct mgoutil_test.structWithDupKeys`,
}, {
	obj:         &inlineMap{A: 1, M: map[string]interface{}{"a": 1}},
	expectError: `Can't have key "a" in inlined map; conflicts with struct field`,
}, {
	obj:         &bson.Raw{0x0a, []byte{}},
	expectError: `cannot marshal: Attempted to marshal Raw kind 10 as a document`,
}, {
	obj:         99,
	expectError: `cannot marshal: Can't marshal int as a BSON document`,
}, {
	obj: struct {
		Id string `bson:"_id"`
		A  int
	}{"hello", 1},
	expect: bson.Update{
		Set: bson.M{
			"a": 1,
		},
		Unset: bson.M{},
	},
}, {
	obj: map[string]string{
		"_id": "hello",
		"a":   "goodbye",
	},
	expect: bson.Update{
		Set: bson.M{
			"a": "goodbye",
		},
		Unset: bson.M{},
	},
}, {
	obj:         map[int]string{34: "hello"},
	expectError: `map key not a string`,
}}

func (*S) TestAsUpdate(c *gc.C) {
	for _, test := range asUpdateTests {
		u, err := bson.AsUpdate(test.obj)
		if test.expectError != "" {
			c.Assert(err, gc.ErrorMatches, test.expectError)
		} else {
			c.Assert(err, gc.Equals, nil)
			c.Assert(u, gc.DeepEquals, test.expect)
		}
	}
}

type typeWithGetter struct {
	result interface{}
	err    error
}

func (t *typeWithGetter) GetBSON() (interface{}, error) {
	if t == nil {
		return "<value is nil>", nil
	}
	return t.result, t.err
}

type inlineInt struct {
	V struct{ A, B int } ",inline"
}
type inlineMap struct {
	A int
	M map[string]interface{} ",inline"
}

type structWithDupKeys struct {
	Name  byte
	Other byte "name" // Tag should precede.
}

// Wrap up the document elements contained in data, prepending the int32
// length of the data, and appending the '\x00' value closing the document.
func wrapInDoc(data string) string {
	result := make([]byte, len(data)+5)
	binary.LittleEndian.PutUint32(result, uint32(len(result)))
	copy(result[4:], []byte(data))
	return string(result)
}
