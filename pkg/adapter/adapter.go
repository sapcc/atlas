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
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/targetgroup"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type customSD struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels"`
}

// Adapter runs service discovery implementation and converts its target groups
// to JSON and writes to a k8s configmap.
type Adapter struct {
	ctx          context.Context
	disc         discovery.Discoverer
	groups       map[string]*customSD
	manager      *discovery.Manager
	output       string
	configMap    string
	outputTarget string
	namespace    string
	name         string
	logger       log.Logger
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
func (a *Adapter) generateTargetGroups(allTargetGroups map[string][]*targetgroup.Group) {
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
		if a.outputTarget == "configmap" {
			err := a.writeConfigMap()
			if err != nil {
				level.Error(log.With(a.logger, "component", "sd-adapter")).Log("err", err)
			}
		} else if a.outputTarget == "file" {
			err := a.writeFile()
			if err != nil {
				level.Error(log.With(a.logger, "component", "sd-adapter")).Log("err", err)
			}
		} else {
			level.Error(log.With(a.logger, "component", "sd-adapter")).Log("err", "No ouput (file or configmap) choosen")
		}
	}

}

// Writes JSON formatted targets to file
func (a *Adapter) writeFile() error {
	arr := mapToArray(a.groups)
	b, _ := json.MarshalIndent(arr, "", "    ")

	dir, _ := filepath.Split(a.output)
	tmpfile, err := ioutil.TempFile(dir, "ipmi-sd")
	if err != nil {
		return err
	}
	defer tmpfile.Close()

	_, err = tmpfile.Write(b)
	if err != nil {
		return err
	}

	err = os.Rename(tmpfile.Name(), a.output)
	if err != nil {
		return err
	}
	return nil
}

// Writes JSON formatted targets to a file in a configmap
func (a *Adapter) writeConfigMap() error {
	arr := mapToArray(a.groups)
	b, _ := json.MarshalIndent(arr, "", "    ")

	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}
	configMap, err := clientset.CoreV1().ConfigMaps(a.namespace).Get(a.configMap, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if configMap.Data == nil {
		configMap.Data = make(map[string]string)
	}
	configMap.Data[a.output] = string(b)

	level.Debug(log.With(a.logger, "component", "sd-adapter")).Log("info", fmt.Sprintf("writing targets to configmap: %s, in namespace: %s", a.configMap, a.namespace))
	configMap, err = clientset.CoreV1().ConfigMaps(a.namespace).Update(configMap)
	if err != nil {
		return err
	}

	return nil
}

func (a *Adapter) runCustomSD(ctx context.Context) {
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
			a.generateTargetGroups(allTargetGroups)
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
func NewAdapter(ctx context.Context, fileName string, name string, d discovery.Discoverer, configMap string, outputTarget string, namespace string, logger log.Logger) *Adapter {
	return &Adapter{
		ctx:          ctx,
		disc:         d,
		groups:       make(map[string]*customSD),
		manager:      discovery.NewManager(ctx, logger),
		output:       fileName,
		name:         name,
		configMap:    configMap,
		outputTarget: outputTarget,
		namespace:    namespace,
		logger:       logger,
	}
}
