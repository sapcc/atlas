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
	"io/ioutil"
	"os"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/sapcc/atlas/pkg/util"
)

type File struct {
	fileName string
	logger   log.Logger
	data     string
}

func NewFile(fileName string, logger log.Logger) (f *File, err error) {
	return &File{
		fileName: fileName,
		logger:   logger,
	}, err
}

func (c *File) GetData(name string) (data string, err error) {
	d, err := ioutil.ReadFile(c.fileName)
	if os.IsNotExist(err) {
		level.Debug(log.With(c.logger, "component", "writer")).Log("debug", fmt.Sprintf("targetsfile does not exist"))
		return data, nil
	}
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

func (c *File) loadData() (err error) {
	d, err := ioutil.ReadFile(c.fileName)
	if err != nil {
		return
	}
	data := string(d)
	data = strings.TrimSuffix(data, "\n")
	c.data = data

	return
}

// Writes string data to configmap.
func (c *File) Write(name, data string) (err error) {
	err = util.RetryOnConflict(util.DefaultBackoff, func() (err error) {
		level.Debug(log.With(c.logger, "component", "writer")).Log("debug", fmt.Sprintf("writing targets to file: %s", c.fileName))
		err = ioutil.WriteFile(c.fileName, []byte(data), 0644)
		return err
	})
	return err
}

func split(r rune) bool {
	return r == '=' || r == ';'
}
