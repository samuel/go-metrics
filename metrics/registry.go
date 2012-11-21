package metrics

import (
	"sync"
)

type Registry struct {
	scope   string
	metrics map[string]interface{}
	mutex   sync.RWMutex
}

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

func (r *Registry) Do(f func(name string, metric interface{}) error) error {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for name, metric := range r.metrics {
		if err := f(name, metric); err != nil {
			return err
		}
	}

	return nil
}
