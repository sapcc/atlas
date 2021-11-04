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
	"sync"

	"github.com/go-kit/kit/log"
	promDiscovery "github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/sapcc/atlas/pkg/writer"
)

type Status struct {
	sync.Mutex
	Up bool
}

type AdapterFactory func(ctx context.Context, m *promDiscovery.Manager, w writer.Writer, logger log.Logger) (Adapter, error)

type Adapter interface {
	GetStatus() *Status
	Run(ctx context.Context, updates <-chan []*targetgroup.Group)
	GetNumberOfTargetsFor(label string) int
	GetData() (string, error)
}
