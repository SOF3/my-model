/*
 * Poggit
 *
 * Copyright (C) 2018 Poggit
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

package myModel

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
)

func Generate(config GeneratorConfig, seeds []reflect.Type) error {
	config.WriteGo("package " + config.Package + "\n")

	schema := &Schema{
		Tables: map[string]*MainTable{},
	}

	for _, seed := range seeds {
		if seed.Kind() != reflect.Ptr || seed.Elem().Kind() != reflect.Struct {
			return errors.New("seeds must be pointers to structs")
		}
		schema.getTable(seed.Elem())
	}

	incomplete := true
	for incomplete {
		incomplete = false
		for _, table := range schema.Tables {
			if !table.yielded {
				incomplete = true
				if err := schema.yieldTable(table); err != nil {
					return err
				}
			}
		}
	}

	if err := schema.computeEdges(); err != nil {
		return err
	}

	schema.OutputSql(config)

	// file, err := os.Create("schema.json")
	// if err != nil {
	// 	return err
	// }
	// defer file.Close()

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "\t")
	err := encoder.Encode(schema)
	if err != nil {
		return err
	}

	return nil
}
