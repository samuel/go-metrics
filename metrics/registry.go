package metrics

import (
	"sync"
)

type Registry struct {
	scope   string
	metrics map[string]interface{}
	mutex   sync.RWMutex
}

type Collection interface {
	Metrics() map[string]interface{}
}

type Doer func(name string, metric interface{}) error

func NewRegistry() *Registry {
	return &Registry{
		metrics: make(map[string]interface{}),
	}
}

func (r *Registry) scopedName(name string) string {
	if r.scope != "" {
		return r.scope + "/" + name
	}
	return name
}

func (r *Registry) Scope(scope string) *Registry {
	return &Registry{
		scope:   r.scopedName(scope),
		metrics: r.metrics,
	}
}

func (r *Registry) Add(name string, metric interface{}) {
	r.mutex.Lock()
	r.metrics[r.scopedName(name)] = metric
	r.mutex.Unlock()
}

func (r *Registry) Remove(name string) {
	r.mutex.Lock()
	delete(r.metrics, r.scopedName(name))
	r.mutex.Unlock()
}

func (r *Registry) Do(f Doer) error {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return do("", r.metrics, f)
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
