/**
 * Copyright 2019 SAP SE
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/sapcc/atlas/pkg/writer"
)

type customSD struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels"`
}

// Adapter runs service discovery implementation and converts its target groups
// to JSON and writes to a k8s configmap.
type Prom struct {
	writer         writer.Writer
	ctx            context.Context
	groups         map[string]*customSD
	outputFileName string
	logger         log.Logger
	Status         *Status
}

// New creates a new instance of Adapter.
func NewPrometheus(ctx context.Context, outputFileName string, w writer.Writer, logger log.Logger) Adapter {

	return &Prom{
		ctx:            ctx,
		groups:         make(map[string]*customSD),
		writer:         w,
		outputFileName: outputFileName,
		Status:         &Status{Up: false},
		logger:         logger,
	}
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
func (p *Prom) generateTargetGroups(allTargetGroups []*targetgroup.Group) error {
	tempGroups := make(map[string]*customSD)
	for i, group := range allTargetGroups {
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
		key := fmt.Sprintf("%s:%d", group.Source, i)
		tempGroups[key] = &customSD{
			Targets: newTargets,
			Labels:  newLabels,
		}
	}
	if !reflect.DeepEqual(p.groups, tempGroups) {
		p.groups = tempGroups
		err := p.writeOutput()
		if err != nil {
			level.Error(log.With(p.logger, "component", "sd-adapter")).Log("err", err)
			return err
		}
	}

	return nil

}

func (p *Prom) writeOutput() error {
	arr := mapToArray(p.groups)
	//b, _ := json.MarshalIndent(arr, "", "")
	b, _ := json.Marshal(arr)

	return p.writer.Write(p.outputFileName, string(b))
}

func (p *Prom) GetStatus() *Status {
	return p.Status
}

func (p *Prom) GetNumberOfTargetsFor(label string) (targets int) {
	data, err := p.writer.GetData(p.outputFileName)
	if err != nil {
		return targets
	}
	groups := make([]customSD, 0)
	err = json.Unmarshal([]byte(strings.Trim(data, "\"")), &groups)
	for _, g := range groups {
		if val, ok := g.Labels["metrics_label"]; ok {
			if val == label {
				targets++
			}
		}
	}
	return targets
}

func (p *Prom) Run(ctx context.Context, updates <-chan []*targetgroup.Group) {
	data, err := p.writer.GetData(p.outputFileName)
	if err != nil {
		level.Error(log.With(p.logger, "component", "sd-adapter")).Log("err", err)
		return
	}
	// write dummy if configMap is not empty, so that when the node list is empty deepEqual will be false and
	// an empty node list is written to configMap
	if len(data) > 0 {
		p.groups["dummy"] = &customSD{}
	}

	for {
		select {
		case <-ctx.Done():
		case allTargetGroups, ok := <-updates:
			// Handle the case that a target provider exits and closes the channel
			// before the context is done.
			if !ok {
				return
			}
			err := p.generateTargetGroups(allTargetGroups)
			p.Status.Lock()
			p.Status.Up = err == nil
			p.Status.Unlock()
		}
	}
}
