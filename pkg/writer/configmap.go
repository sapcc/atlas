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

package writer

import (
	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/sapcc/atlas/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
)

type ConfigMap struct {
	client    *kubernetes.Clientset
	configMap string
	logger    log.Logger
	ns        string
}

func NewConfigMap(cmName, namespace string, logger log.Logger) (cw *ConfigMap, err error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return cw, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return cw, err
	}

	return &ConfigMap{
		ns:        namespace,
		client:    clientset,
		configMap: cmName,
		logger:    logger,
	}, err
}

func (c *ConfigMap) getConfigMap() (*v1.ConfigMap, error) {
	configMap, err := c.client.CoreV1().ConfigMaps(c.ns).Get(c.configMap, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if configMap.Data == nil {
		configMap.Data = make(map[string]string)
	}

	return configMap, nil

}

func (c *ConfigMap) GetData(name string) (data string, err error) {
	configMap, err := c.getConfigMap()
	if err != nil {
		return data, level.Error(log.With(c.logger, "component", "sd-adapter")).Log("err", err)
	}
	return configMap.Data[name], err
}

// Writes string data to configmap.
func (c *ConfigMap) Write(name, data string) (err error) {
	err = util.RetryOnConflict(util.DefaultBackoff, func() (err error) {
		configMap, err := c.getConfigMap()
		if err != nil {
			return err
		}
		configMap.Data[name] = string(data)

		level.Debug(log.With(c.logger, "component", "sd-adapter")).Log("debug", fmt.Sprintf("writing targets to configmap: %s, file_name: %s, in namespace: %s", c.configMap, name, c.ns))
		configMap, err = c.client.CoreV1().ConfigMaps(c.ns).Update(configMap)
		return err
	})

	return err
}
