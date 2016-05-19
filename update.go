// Copyright (c) 2016 - Canonical Ltd
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

// Package mgoutil provides helper functions relating to the mgo package.
package mgoutil

import (
	"fmt"
	"reflect"

	"gopkg.in/errgo.v1"
	"gopkg.in/mgo.v2/bson"
)

// Update represents a document update operation. When marshaled and
// provided to an update operation, it will set all the fields in Set
// and unset all the fields in Unset.
type Update struct {
	// Set holds the fields to be set keyed by field name.
	Set map[string]interface{} `bson:"$set,omitempty"`

	// Unset holds the fields to be unset keyed by field name. Note that
	// the key values will be ignored.
	Unset map[string]interface{} `bson:"$unset,omitempty"`
}

// AsUpdate returns the given object as an Update value holding all the
// fields of x, which must be acceptable to bson.Marshal, with
// zero-valued omitempty fields returned in Unset and others returned in
// Set. On success, the returned Set and Unset fields will always
// be non-nil, even when they contain no items.
//
// Note that the _id field is omitted, as it is not possible to set this
// in an update operation.
//
// This can be useful where an update operation is required to update
// only some subset of a given document without hard-coding all the
// struct fields into the update document.
//
// For example,
//
//	u, err := AsUpdate(x)
//	if err != nil {
//		...
//	}
//	coll.UpdateId(id, u)
//
// is equivalent to:
//
//	coll.UpdateId(id, x)
//
// as long as all the fields in the database document are
// mentioned in x. If there are other fields stored, they won't
// be affected.
func AsUpdate(x interface{}) (Update, error) {
	v := reflect.ValueOf(x)
	for {
		if vi, ok := v.Interface().(bson.Getter); ok {
			getv, err := vi.GetBSON()
			if err != nil {
				return Update{}, fmt.Errorf("GetBSON failed: %v", err)
				panic(err)
			}
			v = reflect.ValueOf(getv)
			continue
		}
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
			continue
		}
		break
	}
	if v.Type() == typeRaw {
		return nonStructAsUpdate(v.Addr())
	}
	var u Update
	var err error
	switch t := v.Type(); t.Kind() {
	case reflect.Map:
		u, err = mapAsUpdate(v)
	case reflect.Struct:
		u, err = structAsUpdate(v)
	default:
		u, err = nonStructAsUpdate(v)
	}
	return u, err
}

func structAsUpdate(v reflect.Value) (Update, error) {
	sinfo, err := getStructInfo(v.Type())
	if err != nil {
		return Update{}, err
	}
	u := Update{
		Set:   make(bson.M),
		Unset: make(bson.M),
	}
	if sinfo.InlineMap >= 0 {
		if m := v.Field(sinfo.InlineMap); m.Len() != 0 {
			for _, k := range m.MapKeys() {
				ks := k.String()
				if _, found := sinfo.FieldsMap[ks]; found {
					return Update{}, errgo.Newf("Can't have key %q in inlined map; conflicts with struct field", ks)
				}
				if ks != "_id" {
					u.Set[ks] = m.MapIndex(k).Interface()
				}
			}
		}
	}
	var value reflect.Value
	for _, info := range sinfo.FieldsList {
		if info.Key == "_id" {
			continue
		}
		if info.Inline == nil {
			value = v.Field(info.Num)
		} else {
			value = v.FieldByIndex(info.Inline)
		}
		if info.OmitEmpty && isZero(value) {
			u.Unset[info.Key] = nil
		} else {
			u.Set[info.Key] = value.Interface()
		}
	}
	return u, nil
}

func nonStructAsUpdate(v reflect.Value) (Update, error) {
	var m map[string]bson.Raw
	data, err := bson.Marshal(v.Interface())
	if err != nil {
		return Update{}, errgo.Notef(err, "cannot marshal")
	}
	if err := bson.Unmarshal(data, &m); err != nil {
		return Update{}, err
	}
	return mapAsUpdate(reflect.ValueOf(m))
}

func mapAsUpdate(v reflect.Value) (Update, error) {
	if v.Type().Key().Kind() != reflect.String {
		return Update{}, errgo.Newf("map key not a string")
	}
	u := Update{
		Set:   make(bson.M),
		Unset: make(bson.M),
	}
	for _, k := range v.MapKeys() {
		ks := k.String()
		if ks == "_id" {
			continue
		}
		u.Set[ks] = v.MapIndex(k).Interface()
	}
	return u, nil
}
