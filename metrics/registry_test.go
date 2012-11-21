package metrics

import (
	"fmt"
	"reflect"
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
