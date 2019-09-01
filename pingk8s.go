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
	"fmt"
	"time"

	"github.com/sparrc/go-ping"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

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
	for {
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
	}
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
		time.Sleep(3 * time.Second)
	}
}
