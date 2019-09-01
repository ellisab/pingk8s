/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Note: the example only works with the code within the same release/branch.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sparrc/go-ping"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	addr = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
)

var (
	pingDurations = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "ping_durations_seconds",
			Help:       "ping latency distributions.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"service"},
	)
)

func init() {
	prometheus.MustRegister(pingDurations)
	prometheus.MustRegister(prometheus.NewBuildInfoCollector())
}

func main() {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	pods, err := clientset.CoreV1().Pods("").List(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))

	for _, pod := range pods.Items {
		if !pod.Spec.HostNetwork {
			go pinger(pod.GetName(), pod.Status.PodIP)
		}
	}

	// Expose the registered metrics via HTTP.
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func pinger(name, ip string) {
	for {
		pinger, err := ping.NewPinger(ip)
		if err != nil {
			fmt.Printf("ping to %v failed: %v", ip, err)
		}
		pinger.SetPrivileged(true)
		pinger.Count = 1
		pinger.Run()
		stats := pinger.Statistics()
		fmt.Printf("%s %s %v\n", name, ip, stats)
		pingDurations.WithLabelValues(fmt.Sprintf("%v %v MinRtt", name, ip)).Observe(float64(stats.MinRtt))
		pingDurations.WithLabelValues(fmt.Sprintf("%v %v MaxRtt", name, ip)).Observe(float64(stats.MaxRtt))
		pingDurations.WithLabelValues(fmt.Sprintf("%v %v PacketsSent", name, ip)).Observe(float64(stats.PacketsSent))
		pingDurations.WithLabelValues(fmt.Sprintf("%v %v PacketsRecv", name, ip)).Observe(float64(stats.PacketsRecv))
		pingDurations.WithLabelValues(fmt.Sprintf("%v %v PacketLoss", name, ip)).Observe(float64(stats.PacketLoss))
		time.Sleep(3 * time.Second)
	}
}
