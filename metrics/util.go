package metrics

import ( 
	"math"
	"time"
)

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func almostEqual(a, b, diff float64) bool {
	return math.Abs(a-b) < diff
}

// Wait for condition (defined by func) to be true, a utility to create a ticker 
// checking back every 50 ms to see if something (the supplied check func) is done
//
//   WaitFor(func() bool {
//      return ctr.Ct == 0
//   },10)
// timeout (in seconds) is the last arg
func WaitFor(check func() bool, timeoutSecs int) {
	timer := time.NewTicker(50 * time.Millisecond)
	tryct := 0
	timerloop: for _ = range timer.C {
		if check() {
			timer.Stop()
			break timerloop
		}
		if tryct >= timeoutSecs * 20 {   //20 = 1s/50 ms
			timer.Stop()
			break timerloop
		}
		tryct++
	}
}
