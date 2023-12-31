// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package schemas

import (
	"fmt"
	"os"
	"strings"

	"github.com/gitbundle/modules/json"
	"github.com/gitbundle/modules/log"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v2"
)

// Load project data from file, with optional validation
func Load(filename string, data interface{}, validation bool) error {
	isJSON := strings.HasSuffix(filename, ".json")

	bs, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	if validation {
		err := validate(bs, data, isJSON)
		if err != nil {
			return err
		}
	}
	return unmarshal(bs, data, isJSON)
}

func unmarshal(bs []byte, data interface{}, isJSON bool) error {
	if isJSON {
		return json.Unmarshal(bs, data)
	}
	return yaml.Unmarshal(bs, data)
}

func getSchema(filename string) (*jsonschema.Schema, error) {
	c := jsonschema.NewCompiler()
	c.LoadURL = openSchema
	return c.Compile(filename)
}

func validate(bs []byte, datatype interface{}, isJSON bool) error {
	var v interface{}
	err := unmarshal(bs, &v, isJSON)
	if err != nil {
		return err
	}
	if !isJSON {
		v, err = toStringKeys(v)
		if err != nil {
			return err
		}
	}

	var schemaFilename string
	switch datatype := datatype.(type) {
	case *[]*Issue:
		schemaFilename = "issue.json"
	case *[]*Milestone:
		schemaFilename = "milestone.json"
	default:
		return fmt.Errorf("file_format:validate: %T has not a validation implemented", datatype)
	}

	sch, err := getSchema(schemaFilename)
	if err != nil {
		return err
	}
	err = sch.Validate(v)
	if err != nil {
		log.Error("migration validation with %s failed for\n%s", schemaFilename, string(bs))
	}
	return err
}

func toStringKeys(val interface{}) (interface{}, error) {
	var err error
	switch val := val.(type) {
	case map[interface{}]interface{}:
		m := make(map[string]interface{})
		for k, v := range val {
			k, ok := k.(string)
			if !ok {
				return nil, fmt.Errorf("found non-string key %T %s", k, k)
			}
			m[k], err = toStringKeys(v)
			if err != nil {
				return nil, err
			}
		}
		return m, nil
	case []interface{}:
		l := make([]interface{}, len(val))
		for i, v := range val {
			l[i], err = toStringKeys(v)
			if err != nil {
				return nil, err
			}
		}
		return l, nil
	default:
		return val, nil
	}
}
