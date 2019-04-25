package writer

import (
	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/sapcc/ipmi_sd/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
)

type ConfigMap struct {
	client    *kubernetes.Clientset
	configMap string
	fileName  string
	logger    log.Logger
	ns        string
}

func NewConfigMap(cmName, fileName, namespace string, logger log.Logger) (cw *ConfigMap, err error) {
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
		fileName:  fileName,
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

func (c *ConfigMap) GetData() (data string, err error) {
	configMap, err := c.getConfigMap()
	if err != nil {
		return data, level.Error(log.With(c.logger, "component", "sd-adapter")).Log("err", err)
	}
	return configMap.Data[c.fileName], err
}

// Writes string data to configmap.
func (c *ConfigMap) Write(data string) (err error) {
	err = util.RetryOnConflict(util.DefaultBackoff, func() (err error) {
		configMap, err := c.getConfigMap()
		if err != nil {
			return err
		}
		configMap.Data[c.fileName] = string(data)

		level.Debug(log.With(c.logger, "component", "sd-adapter")).Log("debug", fmt.Sprintf("writing targets to configmap: %s, in namespace: %s", c.configMap, c.ns))
		configMap, err = c.client.CoreV1().ConfigMaps(c.ns).Update(configMap)
		return err
	})

	return err
}
