package metrics

import "log"

type NamedValue struct {
	Name  string
	Value float64
}

type NamedDistribution struct {
	Name  string
	Value DistributionValue
}

type RegistrySnapshot struct {
	Values        []NamedValue
	Distributions []NamedDistribution

	resetOnSnapshot bool
	counterValues   map[string]uint64
}

func NewRegistrySnapshot(resetOnSnapshot bool) *RegistrySnapshot {
	return &RegistrySnapshot{
		resetOnSnapshot: resetOnSnapshot,
		counterValues:   make(map[string]uint64),
	}
}

func (rs *RegistrySnapshot) Snapshot(registry Registry) {
	rs.Values = rs.Values[:0]
	rs.Distributions = rs.Distributions[:0]
	registry.Do(func(name string, metric interface{}) error {
		switch m := metric.(type) {
		case *EWMA:
			rs.Values = append(rs.Values, NamedValue{Name: name, Value: m.Rate()})
		case *EWMAGauge:
			rs.Values = append(rs.Values, NamedValue{Name: name, Value: m.Mean()})
		case *Meter:
			rs.Values = append(rs.Values,
				NamedValue{Name: name + "/1m", Value: m.OneMinuteRate()},
				NamedValue{Name: name + "/5m", Value: m.FiveMinuteRate()},
				NamedValue{Name: name + "/15m", Value: m.FifteenMinuteRate()},
			)
		case Histogram:
			v := m.Distribution()
			if v.Count > 0 {
				perc := m.Percentiles(DefaultPercentiles)
				m.Clear()
				rs.Distributions = append(rs.Distributions, NamedDistribution{Name: name, Value: v})
				for i, p := range perc {
					rs.Values = append(rs.Values, NamedValue{
						Name:  name + "/" + DefaultPercentileNames[i],
						Value: float64(p),
					})
				}
			}
		case *Counter:
			if rs.resetOnSnapshot {
				rs.Values = append(rs.Values, NamedValue{Name: name, Value: float64(m.Reset())})
			} else {
				oldValue := rs.counterValues[name]
				newValue := m.Count()
				rs.counterValues[name] = newValue
				delta := newValue
				if newValue >= oldValue {
					delta = newValue - oldValue
				}
				rs.Values = append(rs.Values, NamedValue{Name: name, Value: float64(delta)})
			}
		case CounterMetric:
			oldValue := rs.counterValues[name]
			newValue := m.Count()
			rs.counterValues[name] = newValue
			delta := newValue
			if newValue >= oldValue {
				delta = newValue - oldValue
			}
			rs.Values = append(rs.Values, NamedValue{Name: name, Value: float64(delta)})
		case GaugeMetric:
			rs.Values = append(rs.Values, NamedValue{Name: name, Value: m.Value()})
		case DistributionMetric:
			rs.Distributions = append(rs.Distributions, NamedDistribution{Name: name, Value: m.Value()})
		default:
			log.Printf("metrics.RegistrySnapshot: unrecognized metric type for %s: %T %+v", name, m, m)
		}
		return nil
	})
}

func (rs *RegistrySnapshot) Scope(scope string) Registry {
	panic("Scope called on RegistrySnapshot")
}

func (rs *RegistrySnapshot) Add(name string, metric interface{}) {
	panic("Add called on RegistrySnapshot")
}

func (rs *RegistrySnapshot) Remove(name string) {
	panic("Remove called on RegistrySnapshot")
}

func (rs *RegistrySnapshot) Do(f Doer) error {
	for _, v := range rs.Values {
		if err := f(v.Name, GaugeValue(v.Value)); err != nil {
			return err
		}
	}
	for _, v := range rs.Distributions {
		if err := f(v.Name, v); err != nil {
			return err
		}
	}
	return nil
}
