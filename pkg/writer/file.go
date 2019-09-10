package writer

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/sapcc/atlas/pkg/util"
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
	d, err := ioutil.ReadFile(c.fileName)
	data = string(d)
	data = strings.TrimSuffix(data, "\n")
	files := strings.FieldsFunc(data, split)

	for i, f := range files {
		t := strings.TrimSuffix(strings.Trim(strings.TrimSpace(f), "\""), "\n")
		t = strings.Replace(t, "\n", "", -1)
		if t == name {
			return files[i+1], err
		}
	}

	return data, err
}

// Writes string data to configmap.
func (c *File) Write(name, data string) (err error) {
	err = util.RetryOnConflict(util.DefaultBackoff, func() (err error) {

		c.data[name] = string(data) + ";"
		b := new(bytes.Buffer)
		for key, value := range c.data {
			fmt.Fprintf(b, "%s=\"%s\"\n", key, value)
		}

		level.Debug(log.With(c.logger, "component", "writer")).Log("debug", fmt.Sprintf("writing targets to file: %s", name))
		err = ioutil.WriteFile(c.fileName, b.Bytes(), 0644)
		return err
	})

	return err
}

func split(r rune) bool {
	return r == '=' || r == ';'
}
