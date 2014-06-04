// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package metrics

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sync"
)

type Registry interface {
	Scope(scope string) Registry
	Add(name string, metric interface{})
	Remove(name string)
	Do(f Doer) error
}

type registry struct {
	scope   string
	metrics map[string]interface{}
	mutex   sync.RWMutex
}

type filteredRegistry struct {
	registry Registry
	include  []*regexp.Regexp
	exclude  []*regexp.Regexp
}

type Collection interface {
	Metrics() map[string]interface{}
}

type Doer func(name string, metric interface{}) error

// Registry

func NewRegistry() Registry {
	return &registry{
		metrics: make(map[string]interface{}),
	}
}

func (r *registry) scopedName(name string) string {
	if r.scope != "" {
		return r.scope + "/" + name
	}
	return name
}

func (r *registry) Scope(scope string) Registry {
	return &registry{
		scope:   r.scopedName(scope),
		metrics: r.metrics,
	}
}

func (r *registry) Add(name string, metric interface{}) {
	r.mutex.Lock()
	r.metrics[r.scopedName(name)] = metric
	r.mutex.Unlock()
}

func (r *registry) Remove(name string) {
	r.mutex.Lock()
	delete(r.metrics, r.scopedName(name))
	r.mutex.Unlock()
}

func (r *registry) Do(f Doer) error {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return do("", r.metrics, f)
}

// FilteredRegistry

func NewFilterdRegistry(registry Registry, include []*regexp.Regexp, exclude []*regexp.Regexp) Registry {
	return &filteredRegistry{registry, include, exclude}
}

func (r *filteredRegistry) Do(f Doer) error {
	return r.registry.Do(func(name string, metric interface{}) error {
		if r.exclude != nil {
			for _, re := range r.exclude {
				if re.MatchString(name) {
					return nil
				}
			}
		}
		if r.include == nil {
			return f(name, metric)
		}
		for _, re := range r.include {
			if re.MatchString(name) {
				return f(name, metric)
			}
		}
		return nil
	})
}

func (r *filteredRegistry) Scope(scope string) Registry {
	return &filteredRegistry{r.registry.Scope(scope), r.include, r.exclude}
}

func (r *filteredRegistry) Add(name string, metric interface{}) {
	r.registry.Add(name, metric)
}

func (r *filteredRegistry) Remove(name string) {
	r.registry.Remove(name)
}

func do(scope string, metrics map[string]interface{}, f Doer) error {
	for name, metric := range metrics {
		if scope != "" {
			name = scope + "/" + name
		}
		if collection, ok := metric.(Collection); ok {
			met := collection.Metrics()
			if met != nil {
				if err := do(name, met, f); err != nil {
					return err
				}
			}
		} else if err := f(name, metric); err != nil {
			return err
		}
	}
	return nil
}

func RegistryHandler(reg Registry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprintf(w, "{\n")
		first := true
		enc := json.NewEncoder(w)
		reg.Do(func(name string, metric interface{}) error {
			if !first {
				fmt.Fprintf(w, ",")
			}
			first = false
			fmt.Fprintf(w, "%q: ", name)
			// Ignore any error since there's not much that can
			// be done at this point since the headers have been sent
			if err := enc.Encode(metric); err != nil {
				log.Printf("metrics: failed to encode metric of type %T: %s", metric, err.Error())
			}
			return nil
		})
		fmt.Fprintf(w, "\n}\n")
	})
}
