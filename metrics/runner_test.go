package metrics

import (
	"testing"
	"time"
	//"log"
)

func TestAddSnapshotReceiver(t *testing.T) {

	// add a receiver
	AddSnapshotReceiver(func(snap Snapshot) {
		//log.Println("Metrics Snapshot ", name, snap.Ts, snap.Counters)
	})
	if len(receivers) != 1 {
		t.Errorf("Should have added Receiver to map%d", len(receivers))
	}

}

func TestReceiverCallback(t *testing.T) {
	var isDone bool
	var snapTest Snapshot
	AddSnapshotReceiver(func(snap Snapshot) {
		snapTest = snap
		isDone = true
	})
	// start metrics:  keep 10 snapshots in history, every 200 milliseconds
	go RunMetricsHeartbeat(10, 500*time.Millisecond)
	//time.Sleep(time.Millisecond * 20)// make sure its running
	IncrCounter("test1")
	IncrCounter("test1")
	IncrCounter("test1")

	WaitFor(func() bool {
		return isDone
	}, 2)

	if ct, ok := snapTest.Ints["test1"]; !ok || ct != 3 {
		t.Errorf("Should have had a count of 3 but was %d", ct)
	}
}

func TestMetricsFunction(t *testing.T) {
	var isDone bool
	var snapTest Snapshot
	AddMetricsFunction(func(snap *Snapshot) {
		snap.Ints["test_metricsx"] = 900
	})
	AddSnapshotReceiver(func(snap Snapshot) {
		snapTest = snap
		isDone = true
	})
	go RunMetricsHeartbeat(10, 500*time.Millisecond)

	WaitFor(func() bool {
		return isDone
	}, 2)

	if ct, ok := snapTest.Ints["test_metricsx"]; !ok || ct != 900 {
		t.Errorf("Should have had a count of 900 but was %d", ct)
	}
}
