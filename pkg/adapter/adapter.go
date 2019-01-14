// Copyright 2018 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"

	"github.com/sapcc/ipmi_sd/pkg/retry"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
)

type customSD struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels"`
}

// Adapter runs service discovery implementation and converts its target groups
// to JSON and writes to a k8s configmap.
type Adapter struct {
	ctx       context.Context
	disc      discovery.Discoverer
	groups    map[string]*customSD
	manager   *discovery.Manager
	cs        *kubernetes.Clientset
	output    string
	configMap string
	namespace string
	name      string
	logger    log.Logger
	Status    *Status
}

type Status struct {
	sync.Mutex
	Up bool
}

func mapToArray(m map[string]*customSD) []customSD {
	arr := make([]customSD, 0, len(m))
	for _, v := range m {
		arr = append(arr, *v)
	}
	return arr
}

// Parses incoming target groups updates. If the update contains changes to the target groups
// Adapter already knows about, or new target groups, we Marshal to JSON and write to file.
func (a *Adapter) generateTargetGroups(allTargetGroups map[string][]*targetgroup.Group) error {
	tempGroups := make(map[string]*customSD)
	for k, sdTargetGroups := range allTargetGroups {
		for i, group := range sdTargetGroups {
			newTargets := make([]string, 0)
			newLabels := make(map[string]string)

			for _, targets := range group.Targets {
				for _, target := range targets {
					newTargets = append(newTargets, string(target))
				}
			}

			for name, value := range group.Labels {
				newLabels[string(name)] = string(value)
			}
			// Make a unique key, including the current index, in case the sd_type (map key) and group.Source is not unique.
			key := fmt.Sprintf("%s:%s:%d", k, group.Source, i)
			tempGroups[key] = &customSD{
				Targets: newTargets,
				Labels:  newLabels,
			}
		}
	}
	if !reflect.DeepEqual(a.groups, tempGroups) {
		a.groups = tempGroups
		err := a.writeOutput()
		if err != nil {
			level.Error(log.With(a.logger, "component", "sd-adapter")).Log("err", err)
			return err
		}
	}

	return nil

}

func (a *Adapter) getConfigMap() (*v1.ConfigMap, error) {
	configMap, err := a.cs.CoreV1().ConfigMaps(a.namespace).Get(a.configMap, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if configMap.Data == nil {
		configMap.Data = make(map[string]string)
	}

	return configMap, nil

}

// Writes JSON formatted targets to configmap.
func (a *Adapter) writeOutput() error {
	arr := mapToArray(a.groups)
	b, _ := json.MarshalIndent(arr, "", "    ")

	err := retry.RetryOnConflict(retry.DefaultBackoff, func() (err error) {
		configMap, err := a.getConfigMap()
		if err != nil {
			return err
		}
		configMap.Data[a.output] = string(b)

		level.Debug(log.With(a.logger, "component", "sd-adapter")).Log("debug", fmt.Sprintf("writing targets to configmap: %s, in namespace: %s", a.configMap, a.namespace))
		configMap, err = a.cs.CoreV1().ConfigMaps(a.namespace).Update(configMap)

		return err
	})
	if err != nil {
		return err
	}
	return nil
}

func (a *Adapter) runCustomSD(ctx context.Context) {
	configMap, err := a.getConfigMap()
	if err != nil {
		level.Error(log.With(a.logger, "component", "sd-adapter")).Log("err", err)
		return
	}
	// write dummy if configMap is not empty, so that when the node list is empty deepEqual will be false and
	// an empty node list is written to configMap
	if len(configMap.Data[a.output]) > 0 {
		a.groups["dummy"] = &customSD{}
	}
	updates := a.manager.SyncCh()
	for {
		select {
		case <-ctx.Done():
		case allTargetGroups, ok := <-updates:
			// Handle the case that a target provider exits and closes the channel
			// before the context is done.
			if !ok {
				return
			}
			err := a.generateTargetGroups(allTargetGroups)
			a.Status.Lock()
			a.Status.Up = err == nil
			a.Status.Unlock()
		}
	}
}

// Run starts a Discovery Manager and the custom service discovery implementation.
func (a *Adapter) Run() {
	go a.manager.Run()
	a.manager.StartCustomProvider(a.ctx, a.name, a.disc)
	go a.runCustomSD(a.ctx)
}

// NewAdapter creates a new instance of Adapter.
func NewAdapter(ctx context.Context, fileName string, name string, d discovery.Discoverer, configMap string, namespace string, logger log.Logger) (*Adapter, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &Adapter{
		ctx:       ctx,
		disc:      d,
		cs:        clientset,
		groups:    make(map[string]*customSD),
		manager:   discovery.NewManager(ctx, logger),
		output:    fileName,
		name:      name,
		configMap: configMap,
		namespace: namespace,
		Status:    &Status{Up: false},
		logger:    logger,
	}, nil
}
