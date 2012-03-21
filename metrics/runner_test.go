package metrics

import (
	"testing"
	"time"
)

func TestAddSnapshotReceiver(t *testing.T) {

	// add a receiver
	AddSnapshotReceiver("60sec", func(name string, snap Snapshot){
	   	//log.Println("Metrics Snapshot ", name, snap.Ts, snap.Counters)
	})
	if len(receivers) != 1 {
		t.Errorf("Should have added Receiver to map%d", len(receivers))
	}
	if recvs, ok:= receivers["60sec"]; !ok || len(recvs) != 1 {
		t.Errorf("Should have added Receiver to list %d", len(recvs))
	}

}

func TestReceiverCallback(t *testing.T) {
	var isDone bool
	var snapTest Snapshot
	AddSnapshotReceiver("1sec", func(name string, snap Snapshot){
		snapTest = snap
		isDone = true
	})
	// start metrics:  keep 10 snapshots in history, every 200 milliseconds
	go RunMetricsHeartbeat("1sec", 10 ,200 * time.Millisecond)
	//time.Sleep(time.Millisecond * 20)// make sure its running
	IncrCounter("test1")
	IncrCounter("test1")
	IncrCounter("test1")

	WaitFor(func() bool {
		return isDone
	},2)

	if ct, ok := snapTest.Counters["test1"]; !ok || ct != 3 {
		t.Errorf("Should have had a count of 3 but was %d", ct)
	}
}

