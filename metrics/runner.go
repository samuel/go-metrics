package metrics

import (
	"sync"
	"time"
)

var (
	metricsMu   sync.Mutex
	recvMu      sync.Mutex
	counters    = make(map[string]uint64)
	counterfs   = make(map[string]float64)
	histograms  = make(map[string]*Histogram)
	snapHistory = make(map[string][]Snapshot)
	receivers   = make(map[string][]MetricsReceiver)
)

type Snapshot struct {
	Counters     map[string]uint64
	Counterfs    map[string]float64
	Ts           int64
	Histograms   map[string]*Histogram
}


type MetricsReceiver func(string,Snapshot)

func History() map[string][]Snapshot {
	metricsMu.Lock()
	history := make(map[string][]Snapshot)
	for n, hist := range snapHistory {
		newSnaps := make([]Snapshot,len(hist))
		for i, snap := range hist {
			newSnaps[i] = snap
		}
		history[n] = newSnaps
	}
	metricsMu.Unlock()
	return history
}

func swapMetrics(name string, maxHistory int, n time.Time) Snapshot {
	metricsMu.Lock()

	snap := Snapshot{
		Counters:counters,
		Counterfs:counterfs,
		Ts:n.UnixNano(),
		Histograms:histograms,
	}

	counters    = make(map[string]uint64)
	counterfs   = make(map[string]float64)
	histograms  = make(map[string]*Histogram)

	for n, hist := range snapHistory {
		if len(hist) > maxHistory {
			snapHistory[n] = hist[1:]
		}
	}

	snapHistory[name] = append(snapHistory[name], snap)
	metricsMu.Unlock()
	return snap
}

// Update Float Counter
func UpdateCounterf(name string, value float64) {
	metricsMu.Lock()
	counterfs[name] += value
	metricsMu.Unlock()
}

// Increment an integer counter
func IncrCounter(name string) {
	metricsMu.Lock()
	counters[name] ++
	metricsMu.Unlock()
}

// Add value to a histogram
func UpdateHistogram(name string, value float64) {
	metricsMu.Lock()
	hist := histograms[name]
	if hist == nil {
		hist = NewUnbiasedHistogram()
		histograms[name] = hist
	}
	hist.Update(value)
	metricsMu.Unlock()
}

// this is a blocking Runner, that starts a heartbeat to 
// take metrics snapshots at specified time, it also calls back
// to any receivers
func RunMetricsHeartbeat(name string, maxHistory int, tick time.Duration) {

	// lets poll back to take metrics snapshots
	timer := time.NewTicker(tick)

	for n := range timer.C {
		snap := swapMetrics(name,maxHistory, n)
		recvMu.Lock()
		recvs := receivers[name]
		for _, recv := range recvs {
			recv(name,snap)
		}
		recvMu.Unlock()
	}
}

// register to get a callback for every Metrics Snapshot
func AddSnapshotReceiver(name string, receiver MetricsReceiver) {
	recvMu.Lock()
	recvs, ok := receivers[name]
	if  !ok {
		recvs = make([]MetricsReceiver,0)
	}
	recvs = append(recvs, receiver)
	receivers[name] = recvs
	recvMu.Unlock()
}