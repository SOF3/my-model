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
	"errors"
	"fmt"
	"github.com/hoop33/go-elvis"
	"reflect"
	"strings"
)

func (schema *Schema) yieldTable(table *MainTable) error {
	table.yielded = true

	if table.Type.Kind() != reflect.Struct {
		return errors.New("invalid table type: not a struct: " + fmt.Sprintf("%v", table.Type))
	}

	for i := 0; i < table.Type.NumField(); i++ {
		field := table.Type.Field(i)

		if strings.IndexRune(field.Name, '_') != -1 {
			return errors.New("field names must not contain underscores to prevent collision with generated columns")
		}

		tag := field.Tag
		fieldType := field.Type

		isSlice := false
		if fieldType.Kind() == reflect.Slice {
			isSlice = true
			fieldType = fieldType.Elem()
		}

		isPointer := false
		if fieldType.Kind() == reflect.Ptr {
			isPointer = true
			fieldType = fieldType.Elem()
		}

		isComplex := !isSimpleStruct(fieldType)

		if isPointer && !isComplex {
			return errors.New("field must not be pointer to non-complex type")
		}

		if _, exists := tag.Lookup("parent"); exists {
			if !isPointer || isSlice || !isComplex {
				return errors.New("parent column must be a pointer to a non-slice complex type")
			}
			// parent reference; the other type must contain this type without pointer
			parent := schema.mustGetTable(fieldType.Name())
			table.Edges = append(table.Edges, &Edge{
				Name:      field.Name,
				PeerTable: fieldType.Name(),
				Type:      elvis.Ternary(parent.FindEdgeByPeerTable(table.Name).Type == EdgeTypeOneMulti, EdgeTypeMultiOneParent, EdgeTypeOneOneParent).(EdgeType),
			})

			keys := schema.mustGetTable(fieldType.Name()).PrimaryKeys
			renamedKeys := make([]string, 0, len(keys))
			for _, key := range keys {
				renamedKeys = append(renamedKeys, fieldType.Name()+"_"+key)
			}
			if _, exists := tag.Lookup("primaryKey"); exists {
				table.PrimaryKeys = append(table.PrimaryKeys, renamedKeys...)
			} else if indexName, exists := tag.Lookup("unique"); exists {
				if _, keyExists := table.UniqueKeys[indexName]; !keyExists {
					table.UniqueKeys[indexName] = make([]string, 0, 1)
				}
				table.UniqueKeys[indexName] = append(table.UniqueKeys[indexName], renamedKeys...)
			} else if indexName, exists := tag.Lookup("composite"); exists {
				if _, keyExists := table.CompositeKeys[indexName]; !keyExists {
					table.CompositeKeys[indexName] = make([]string, 0, 1)
				}
				table.CompositeKeys[indexName] = append(table.CompositeKeys[indexName], renamedKeys...)
			}
		} else if isPointer {
			if isSlice {
				// multi-multi edge, create an anonymous table for storing edges
				table.Edges = append(table.Edges, &Edge{
					Name:      field.Name,
					PeerTable: fieldType.Name(),
					Type:      EdgeTypeMultiMulti,
				})
				schema.getTable(fieldType)
			} else {
				// multi-one edge, an ON DELETE SET NULL foreign key on the other type from this type
				table.Edges = append(table.Edges, &Edge{
					Name:      field.Name,
					PeerTable: fieldType.Name(),
					Type:      EdgeTypeMultiOne,
				})
				schema.getTable(fieldType)
			}
		} else if isComplex { // !isPointer
			childTable := schema.getTable(fieldType)
			if childTable.knownParent != nil {
				return errors.New("there can be only one field (parent) among all types containing another type (child) without pointers; " +
					"both " + childTable.knownParent.Name + " and " + table.Name + " contain " + fieldType.Name())
			}
			schema.getTable(fieldType).knownParent = table
			if isSlice {
				// one-multi edge, this type is parent of the other type
				table.Edges = append(table.Edges, &Edge{
					Name:      field.Name,
					PeerTable: fieldType.Name(),
					Type:      EdgeTypeOneMulti,
				})
			} else {
				// one-one edge, this type is parent of the other type
				table.Edges = append(table.Edges, &Edge{
					Name:      field.Name,
					PeerTable: fieldType.Name(),
					Type:      EdgeTypeOneOne,
				})
			}
		} else if isSlice { // !isPointer && !isComplex
			// create an anonymous table that contain values in this type
			// TODO AuxTables
		} else {
			mysqlType, err := SimpleToMysqlType(fieldType, tag)
			if err != nil {
				return errors.New(err.Error() + " in " + table.Type.Name() + "." + field.Name)
			}
			field := &MysqlField{
				Name: field.Name,
				Type: mysqlType,
			}
			if _, exists := tag.Lookup("nullable"); exists {
				field.Nullable = true
			}
			if _, exists := tag.Lookup("primaryKey"); exists {
				table.PrimaryKeys = append(table.PrimaryKeys, field.Name)
				if _, exists := tag.Lookup("autoIncrement"); exists {
					field.AutoIncrement = true
				}
			} else if indexName, exists := tag.Lookup("unique"); exists {
				if _, keyExists := table.UniqueKeys[indexName]; !keyExists {
					table.UniqueKeys[indexName] = make([]string, 0, 1)
				}
				table.UniqueKeys[indexName] = append(table.UniqueKeys[indexName], field.Name)
			} else if indexName, exists := tag.Lookup("composite"); exists {
				if _, keyExists := table.CompositeKeys[indexName]; !keyExists {
					table.CompositeKeys[indexName] = make([]string, 0, 1)
				}
				table.CompositeKeys[indexName] = append(table.CompositeKeys[indexName], field.Name)
			}
			table.SimpleFields = append(table.SimpleFields, field)
		}
	}
	return nil
}

func isSimpleStruct(p reflect.Type) bool {
	switch p.Kind() {
	case reflect.Bool:
		return true
	case reflect.Int8:
		return true
	case reflect.Int16:
		return true
	case reflect.Int32:
		return true
	case reflect.Int64:
		return true
	case reflect.Uint8:
		return true
	case reflect.Uint16:
		return true
	case reflect.Uint32:
		return true
	case reflect.Uint64:
		return true
	case reflect.Float32:
		return true
	case reflect.Float64:
		return true
	case reflect.String:
		return true
	case reflect.Struct:
		return p.PkgPath() == "time" && p.Name() == "Time"
	}
	return false
}

func SimpleToMysqlType(typ reflect.Type, tag reflect.StructTag) (string, error) {
	switch typ.Kind() {
	case reflect.Bool:
		return "BOOL", nil
	case reflect.Int8:
		return "TINYINT SIGNED", nil
	case reflect.Int16:
		return "SMALLINT SIGNED", nil
	case reflect.Int32:
		return "INT SIGNED", nil
	case reflect.Int64:
		return "BIGINT SIGNED", nil
	case reflect.Uint8:
		return "TINYINT UNSIGNED", nil
	case reflect.Uint16:
		return "SMALLINT UNSIGNED", nil
	case reflect.Uint32:
		return "INT UNSIGNED", nil
	case reflect.Uint64:
		return "BIGINT UNSIGNED", nil
	case reflect.Float32:
		return "FLOAT", nil
	case reflect.Float64:
		return "DOUBLE", nil
	case reflect.String:
		base := "VARCHAR"
		if _, ok := tag.Lookup("fixed"); ok {
			base = "CHAR"
		}
		if value, ok := tag.Lookup("width"); ok {
			return base + "(" + value + ")", nil
		}
		if textType, ok := tag.Lookup("text"); ok {
			if base == "CHAR" {
				return "", errors.New("text columns cannot be fixed")
			}
			switch textType {
			case "tiny":
				return "TINYTEXT", nil
			case "small":
				return "TEXT", nil
			case "":
				return "TEXT", nil
			case "medium":
				return "MEDIUMTEXT", nil
			case "long":
				return "LONGTEXT", nil
			}
			return "", errors.New("unknown text type")
		}

		return "", errors.New("string declaration must either have the width tag")

	case reflect.Struct:
		if typ.PkgPath() == "time" && typ.Name() == "Time" {
			return "TIMESTAMP", nil
		}
	}
	return "", errors.New(fmt.Sprintf("unknown type %v", typ.Kind()))
}
