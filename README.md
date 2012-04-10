go-metrics is a library for aggregating data into metrics (counts, gauges, histograms) as well as a daemon server to collect data and report metrics.  

It can be used to embed in an application for tracking internal app metrics, and then reporting out. 

Or the daemon server can collect data over UDP requests, and send aggregate metrics to [Librato](https://metrics.librato.com/) or [Stathat](http://Stathat.com)  .  (TODO:   Graphite reporter)

See the Metricsd folder for how to send to Stathat/librato.



This is an example of embedding the metrics in your application :

    import "github.com/samuel/go-metrics/metrics"
    import "time"

    func mystartup() {
        // get a snapshot of metrics every heartbeat
        metrics.AddSnapshotReceiver("60sec", func(name string, snap metrics.Snapshot){
            log.Println("Metrics Snapshot ", name, snap.Ts, snap.Counters)
        })
        // start metrics:  keep 10 snapshots in history, every 60 seconds is heartbeat
        go metrics.RunMetricsHeartbeat("60sec", 10 ,60 * time.Second)
    }

    func hello(w http.ResponseWriter, r *http.Request) {
        // update a counter
        metrics.IncrCounter("hello")
        io.WriteString(w, "hello")
    }

    
    func checkCapacity() {
        if snaps,ok := metrics.History()["60sec"]; ok {
            if len(snaps) > 0 {
                // most recent
                snap := snaps[len(snaps) - 1]
                if rate, ok := snap["hello"]; ok {
                    if rate > 10000 {
                        StartNewServer()
                    }
                }
            }
        }
    }

