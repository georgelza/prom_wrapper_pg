/*****************************************************************************
*
*	File			: main.go
*
* 	Created			: 27 March 2023
*
*	Description		: Quick Dirty wrapper for Prometheus (push gateway) and golang library to figure out how to back port it into fs_loader
*
*	Modified		: 29 March 2023	- Start
*
*	By			: George Leonard (georgelza@gmail.com)
*
*
*
*****************************************************************************/

package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

type metrics struct {
	completionTime prometheus.Gauge
	successTime    prometheus.Gauge
	duration       prometheus.Gauge
	records        prometheus.Gauge

	info          *prometheus.GaugeVec
	sql_duration  *prometheus.HistogramVec
	rec_duration  *prometheus.HistogramVec
	api_duration  *prometheus.HistogramVec
	req_processed *prometheus.CounterVec
}

var (

	// We use a registry here to benefit from the consistency checks that
	// happen during registration.
	reg    = prometheus.NewRegistry()
	m      = NewMetrics(reg)
	pusher = push.New("http://127.0.0.1:9091", "pushgateway").Gatherer(reg)
)

func NewMetrics(reg prometheus.Registerer) *metrics {

	m := &metrics{
		///////////////////////////////////////////////////////////////////
		// Example metrics
		completionTime: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "fs_etl_complete_timestamp_seconds",
			Help: "The timestamp of the last completion of a FS ETL job, successful or not.",
		}),

		successTime: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "fs_etl_success_timestamp_seconds",
			Help: "The timestamp of the last successful completion of a FS ETL job.",
		}),

		duration: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "fs_etl_duration_seconds",
			Help: "The duration of the last FS ETL job in seconds.",
		}),

		records: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "fs_etl_records_processed",
			Help: "The number of records processed in the last FS ETL job.",
		}),

		///////////////////////////////////////////////////////////////////
		// My wrapper, for my metrics from my app
		info: prometheus.NewGaugeVec(prometheus.GaugeOpts{ // Shows value, can go up and down
			Name: "txn_count",
			Help: "The number of records discovered to be processed for FS ETL job",
		}, []string{"batch"}),

		//
		sql_duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{ // used to store timed values
			Name: "fs_sql_duration_seconds",
			Help: "Duration of the FS ETL sql requests in seconds",
			// 4 times larger apdex status
			// Buckets: prometheus.ExponentialBuckets(0.1, 1.5, 5),
			// Buckets: prometheus.LinearBuckets(0.1, 5, 15),
			Buckets: []float64{0.1, 0.5, 1, 5, 10, 100},
		}, []string{"batch"}),

		api_duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "fs_api_duration_seconds",
			Help:    "Duration of the FS ETL api requests in seconds",
			Buckets: []float64{0.00001, 0.000015, 0.00002, 0.000025, 0.00003},
		}, []string{"batch"}),

		rec_duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "fs_etl_operations_seconds",
			Help:    "Duration of the entire FS ETL requests in seconds",
			Buckets: []float64{0.001, 0.0015, 0.002, 0.0025, 0.01},
		}, []string{"batch"}),

		req_processed: prometheus.NewCounterVec(prometheus.CounterOpts{ // can only go up/increment, but usefull combined with rate, resets to zero at restart.
			Name: "fs_etl_operations_total",
			Help: "The number of records processed for the FS ETL job.",
		}, []string{"batch"}),
	}

	reg.MustRegister(m.info, m.sql_duration, m.api_duration, m.rec_duration, m.req_processed)

	return m
}

func performBackup() (int, error) {

	// Perform the backup and return the number of backed up records and any
	// applicable error.
	// ...

	rand.Seed(time.Now().UnixNano())
	n := rand.Intn(1000) // if vGeneral.sleep = 1000, then n will be random value of 0 -> 1000  aka 0 and 1 second
	fmt.Printf("API Sleeping %d Millisecond...\n", n)
	time.Sleep(time.Duration(n) * time.Millisecond)

	return 42, nil
}

func mRun() {

	var todo_count = 40

	// simulate a multi second sql query
	sqlstart := time.Now()
	rand.Seed(time.Now().UnixNano())
	n := rand.Intn(10000) // if vGeneral.sleep = 1000, then n will be random value of 0 -> 1000  aka 0 and 1 second (10000 = 10 seconds)
	fmt.Printf("SQL Sleeping %d Millisecond...\n", n)
	time.Sleep(time.Duration(n) * time.Millisecond)

	m.sql_duration.WithLabelValues("eft").Observe(time.Since(sqlstart).Seconds())

	m.info.WithLabelValues("eft").Set(345234523)

	for count := 0; count < todo_count; count++ {
		// Note that successTime is not registered.

		start := time.Now()
		n, err := performBackup() // execute the long running batch job.
		m.records.Set(float64(n)) // How many files back'd up, return variable

		m.api_duration.WithLabelValues("eft").Observe(time.Since(start).Seconds())

		// Note that time.Since only uses a monotonic clock in Go1.9+.
		m.duration.Set(time.Since(start).Seconds()) // execution time = my api_duration
		m.completionTime.SetToCurrentTime()         // last completed time

		if err != nil {
			fmt.Println("DB backup failed:", err)

		} else {
			// Add successTime to pusher only in case of success.
			// We could as well register it with the registry.
			// This example, however, demonstrates that you can
			// mix Gatherers and Collectors when handling a Pusher.

			//pusher.Collector(m.successTime) // as we're inside a loop don't use this otherwise it tries to readd the metric to be collected.
			m.successTime.SetToCurrentTime() // last success time

		}

		// Add is used here rather than Push to not delete a previously pushed
		// success timestamp in case of a failure of this backup.
		if err := pusher.Add(); err != nil {
			fmt.Println("Could not push to Pushgateway:", err)
		}

		rand.Seed(time.Now().UnixNano())
		n = rand.Intn(2000) // if vGeneral.sleep = 1000, then n will be random value of 0 -> 1000  aka 0 and 1 second (2000 = 2 seconds)
		fmt.Printf("Req Sleeping %d Millisecond...\n", n)
		time.Sleep(time.Duration(n) * time.Millisecond)

		m.req_processed.WithLabelValues("eft").Inc()

		m.rec_duration.WithLabelValues("eft").Observe(time.Since(start).Seconds()) // duration for entire loop

		// force a final metric push
		if err := pusher.Add(); err != nil {
			fmt.Println("Could not push to Pushgateway:", err)
		}

	}
}
func main() {

	mRun()

}
