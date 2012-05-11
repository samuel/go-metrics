package metrics

import (
	"sync"
	"time"
)

var (
	metricsMu   sync.Mutex
	snapshot    = NewSnapshot()
	snapHistory = make([]*Snapshot, 0)
	receivers   = make([]MetricsReceiver, 0)
	metfuncs    = make([]MetricsFunction, 0)
)

type Snapshot struct {
	Ints       MetricInt             `json:"ints"`
	Floats     map[string]float64    `json:"floats"`
	Histograms map[string]*Histogram `json:"histograms"`
	Ts         int64                 `json:"ts"`
}

func NewSnapshot() *Snapshot {
	return &Snapshot{
		Ints:       make(MetricInt),
		Floats:     make(MetricFloat),
		Histograms: make(MetricHistogram),
	}
}

type MetricInt map[string]uint64
type MetricFloat map[string]float64
type MetricHistogram map[string]*Histogram
type MetricsReceiver func(Snapshot)
type MetricsFunction func(*Snapshot)

// History grabs a record of Snapshots from memory
func History() []Snapshot {
	metricsMu.Lock()
	newSnaps := make([]Snapshot, len(snapHistory))
	for i, snap := range snapHistory {
		newSnaps[i] = *snap
	}
	metricsMu.Unlock()
	return newSnaps
}

func swapMetrics(maxHistory int, n time.Time) {
	// NOTE: this has no lock, it is in the call to this

	// regardless of if any current metrics for this unit, lets chop one
	if len(snapHistory) > maxHistory {
		snapHistory = snapHistory[1:]
	}

	// if we have collected any metrics, add to history
	snapHistory = append(snapHistory, snapshot)

	snapshot = NewSnapshot()
}

// Update Float Counter
func UpdateCounterf(name string, value float64) {
	metricsMu.Lock()
	snapshot.Floats[name] += value
	metricsMu.Unlock()
}

// Increment an integer counter
func IncrCounter(name string) {
	metricsMu.Lock()
	snapshot.Ints[name]++
	metricsMu.Unlock()
}

// Add value to a histogram
func UpdateHistogram(name string, value float64) {
	metricsMu.Lock()
	hist := snapshot.Histograms[name]
	if hist == nil {
		hist = NewUnbiasedHistogram()
		snapshot.Histograms[name] = hist
	}
	hist.Update(value)
	metricsMu.Unlock()
}

// this is a blocking Runner, that starts a heartbeat to 
// take metrics snapshots at specified time intervals, it also calls back
// to any receivers you have setup
func RunMetricsHeartbeat(maxHistory int, tick time.Duration) {

	timer := time.NewTicker(tick)

	for n := range timer.C {
		metricsMu.Lock()

		curSnap := snapshot
		curSnap.Ts = n.UnixNano()

		for _, mfn := range metfuncs {
			//metricsMu.Unlock() ??
			mfn(curSnap)
			//metricsMu.Lock()
		}

		swapMetrics(maxHistory, n)

		for _, recv := range receivers {
			go recv(*curSnap)
		}

		metricsMu.Unlock()
	}
}

// register to get a callback for every Metrics Snapshot (after it has been taken)
// this is optional, and allows you to view, save, send the metrics::
//		
//		metrics.AddSnapshotReceiver(func(snap metrics.Snapshot){
//			mongdb.Save(snap)
//		})
//		go metrics.RunMetricsHeartbeat(0, time.Second * 60)
//
func AddSnapshotReceiver(receiver MetricsReceiver) {
	metricsMu.Lock()
	receivers = append(receivers, receiver)
	metricsMu.Unlock()
}

// register to get a callback Before every snapshot is taken, and add data.  Instead
// of incrementing values continually, this is good if you already have internal data
// that you want to take a guage on.  You can also calculate derived metrics.::
//		
//		var myVarName float64 = 2000
//		var curUsers int
//		func foo(r http.Request) {
//			myVarName += len(r.Body)
//			metrics.IncrCounter("my_requests")
//		}
//		
//		metrics.AddMetricsFunction(func(snap *metrics.Snapshot){
//			snap.Ints["my_custom_value"] = uint64(myVarName) 
//			snap.Ints["requests_peruser"] = snap.Ints["my_requests"] / curUsers  
//		})
//		
//		go metrics.RunMetricsHeartbeat(0, time.Second * 60)
//
func AddMetricsFunction(metFunc MetricsFunction) {
	metricsMu.Lock()
	metfuncs = append(metfuncs, metFunc)
	metricsMu.Unlock()
}
