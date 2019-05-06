package writer

import (
	"bytes"
	"fmt"
	"io/ioutil"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/sapcc/ipmi_sd/pkg/util"
)

type File struct {
	fileName string
	logger   log.Logger
	data     map[string]string
}

func NewFile(fileName string, logger log.Logger) (f *File, err error) {
	return &File{
		fileName: fileName,
		logger:   logger,
		data:     make(map[string]string),
	}, err
}

func (c *File) GetData(name string) (data string, err error) {
	return c.data[name], err
}

// Writes string data to configmap.
func (c *File) Write(name, data string) (err error) {
	err = util.RetryOnConflict(util.DefaultBackoff, func() (err error) {

		c.data[c.fileName] = string(data)
		b := new(bytes.Buffer)
		for key, value := range c.data {
			fmt.Fprintf(b, "%s=\"%s\"\n", key, value)
		}

		level.Debug(log.With(c.logger, "component", "writer")).Log("debug", fmt.Sprintf("writing targets to file: %s", c.fileName))
		err = ioutil.WriteFile(c.fileName, b.Bytes(), 0644)
		return err
	})

	return err
}
