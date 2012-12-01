// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package metrics

import (
	"fmt"
	"reflect"
	"regexp"
	"testing"
)

func TestRegistry(t *testing.T) {
	r := NewRegistry()
	r2 := r.Scope("test")
	r.Add("num", 1)
	r2.Add("foo", "bar")
	metrics := make(map[string]string)
	r.Do(func(name string, metric interface{}) error {
		metrics[name] = fmt.Sprintf("%+v", metric)
		return nil
	})
	exp := map[string]string{"num": "1", "test/foo": "bar"}
	if !reflect.DeepEqual(metrics, exp) {
		t.Fatalf("registry.Do should have returned %+v instead of %+v", exp, metrics)
	}
}

func TestFilteredRegistry(t *testing.T) {
	r := NewRegistry()
	r.Add("num", 1)
	r.Add("string", "x")

	// nil include and exclude should make the filter a no-op
	fr := NewFilterdRegistry(r, nil, nil)
	out := make(map[string]string)
	fr.Do(func(name string, metric interface{}) error {
		out[name] = fmt.Sprintf("%+v", metric)
		return nil
	})
	exp := map[string]string{"num": "1", "string": "x"}
	if !reflect.DeepEqual(out, exp) {
		t.Fatalf("filteredRegistry.Do should have returned %+v instead of %+v", exp, out)
	}

	// includes
	fr = NewFilterdRegistry(r, []*regexp.Regexp{regexp.MustCompile("^num$")}, nil)
	out = make(map[string]string)
	fr.Do(func(name string, metric interface{}) error {
		out[name] = fmt.Sprintf("%+v", metric)
		return nil
	})
	exp = map[string]string{"num": "1"}
	if !reflect.DeepEqual(out, exp) {
		t.Fatalf("filteredRegistry.Do should have returned %+v instead of %+v", exp, out)
	}

	// excludes
	fr = NewFilterdRegistry(r, nil, []*regexp.Regexp{regexp.MustCompile("^num$")})
	out = make(map[string]string)
	fr.Do(func(name string, metric interface{}) error {
		out[name] = fmt.Sprintf("%+v", metric)
		return nil
	})
	exp = map[string]string{"string": "x"}
	if !reflect.DeepEqual(out, exp) {
		t.Fatalf("filteredRegistry.Do should have returned %+v instead of %+v", exp, out)
	}
}
