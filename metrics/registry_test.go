// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package metrics

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

func TestRegistryHandler(t *testing.T) {
	r := NewRegistry()
	r2 := r.Scope("test")
	r.Add("num", 1)
	r2.Add("foo", "bar")
	r.Add("counter", NewCounter())

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	RegistryHandler(r).ServeHTTP(res, req)
	if res.Code != 200 {
		t.Fatalf("Expected response 200. Got %d", res.Code)
	}
	t.Logf("%s", res.Body.String())
	var out map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
}
