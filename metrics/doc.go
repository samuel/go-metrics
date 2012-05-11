/*
go-metrics is a library for aggregating data into metrics (counts, gauges, histograms) as well as a daemon server to collect data and forward to visualization services.  

It can be used to embed in an application for tracking internal app metrics, and then reporting out. 

Or the daemon server can collect data over UDP requests, and send metrics to [Librato](https://metrics.librato.com/) or [Stathat](http://Stathat.com)  .  

See the Metricsd folder for how to send to Stathat/librato.



Example of usage embedding the metrics in your application ::

    import "github.com/samuel/go-metrics/metrics"
    import "time"

    func init() {
        go metrics.RunMetricsHeartbeat(10 ,60 * time.Second)
    }

    func hello(w http.ResponseWriter, r *http.Request) {
        // update a counter
        metrics.IncrCounter("my_requests")
        metrics.UpdateCounterf("my_requests_size", len(r.Body))
        io.WriteString(w, "hello")
    }


A more elaborate example, registering recievers to get a callback, and custom metrics functions::

    import "github.com/samuel/go-metrics/metrics"
    import "time"

    var myVarName float64 = 2000
    var curUsers int

    func init() {
        metrics.AddSnapshotReceiver(func(snap metrics.Snapshot){
            mongdb.Save(snap)
            checkCapacity(snap)
        })

        metrics.AddMetricsFunction(metricsData)

        go metrics.RunMetricsHeartbeat(0, time.Second * 60)
    }

    func metricsData(snap *metrics.Snapshot) {
        snap.Ints["my_custom_value"] = uint64(myVarName) 
        snap.Ints["requests_peruser"] = snap.Ints["my_requests"] / uint64(curUsers)  
    }
    
    func foo(r http.Request) {
        myVarName += len(r.Body)
        metrics.IncrCounter("my_requests")
    }
    
    // how to use the snapshot data
    func checkCapacity(snap) {
        if rate, ok := snap.Ints["my_requests"]; ok {
            if rate > 10000 {
                StartNewServer()
            }
        }
    }



*/
package metrics
