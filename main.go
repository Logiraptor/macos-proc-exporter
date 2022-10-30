package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/shirou/gopsutil/v3/process"
)

func main() {
	prometheus.MustRegister(&metrics{})
	http.Handle("/metrics", promhttp.Handler())

	port := os.Getenv("PORT")
	if port == "" {
		port = "19002"
	}
	http.ListenAndServe("localhost:"+port, nil)
}

type metrics struct{}

func (m metrics) Describe(ch chan<- *prometheus.Desc) {
	ch <- prometheus.NewDesc("mac_process_cpu_usage_total", "mac_process_cpu_usage", []string{"name", "parent"}, nil)
	ch <- prometheus.NewDesc("mac_process_mem_usage", "mac_process_mem_usage", []string{"name", "parent"}, nil)
}

func (m *metrics) Collect(ch chan<- prometheus.Metric) {
	fmt.Println("Collecting metrics")

	procs, err := process.Processes()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("procs: %d\n", len(procs))

	type resultKey struct {
		name   string
		parent string
		// cmdline string
	}

	type result struct {
		resultKey
		cpuTotal float64
		mem      float32
	}

	var results = make(map[resultKey]result)
	for _, proc := range procs {
		var name, parentName string
		{
			name, err = proc.Name()
			if err != nil {
				log.Println("error getting process name:", err)
				continue
			}
		}

		{
			parentName, err = findParent(name, proc)
			if err != nil {
				log.Println("error getting parent process name:", err)
				continue
			}
		}

		key := resultKey{name, parentName}

		if _, ok := results[key]; !ok {
			results[key] = result{resultKey: key}
		}

		r := results[key]

		{
			times, err := proc.Times()
			if err != nil {
				log.Printf("error getting process cpu for %s: %v", name, err)
				continue
			}

			// Other values are available, but only these are filled in
			totalCPU := times.User + times.System
			r.cpuTotal += totalCPU
		}

		{
			mem, err := proc.MemoryPercent()
			if err != nil {
				log.Printf("error getting process memory for %s: %v", name, err)
				continue
			}

			r.mem += mem
		}

		results[key] = r
	}

	var orderedResults []result
	for _, v := range results {
		orderedResults = append(orderedResults, v)
	}

	for _, r := range orderedResults {
		fmt.Println("name:", r.name, "pct:", r.cpuTotal, "mem:", r.mem)
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("mac_process_cpu_usage_total", "process cpu usage", []string{"name", "parent"}, nil),
			prometheus.CounterValue,
			r.cpuTotal,
			r.name, r.parent,
		)

		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("mac_process_mem_usage", "process memory usage", []string{"name", "parent"}, nil),
			prometheus.GaugeValue,
			float64(r.mem),
			r.name, r.parent,
		)
	}

	fmt.Println("done")
}

func findParent(name string, proc *process.Process) (string, error) {
	var err error
	for {
		proc, err = proc.Parent()
		if err != nil {
			return "", err
		}

		// We've reached the top of the tree
		if proc == nil {
			return name, nil
		}

		parentName, err := proc.Name()
		if err != nil {
			return "", err
		}

		// We've reached the top of the tree, but launchd is in charge of
		// everything so we return the last child name
		if parentName == "launchd" {
			return name, nil
		}

		name = parentName
	}
}
