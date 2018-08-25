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
	"github.com/hoop33/go-elvis"
	"strings"
)

func (schema *Schema) OutputSql(config GeneratorConfig) error {
	for _, table := range schema.getSortedTables() {
		if err := schema.outputSqlMainTable(table, config); err != nil {
			return err
		}
	}

	return nil
}

func (schema *Schema) outputSqlMainTable(table *MainTable, config GeneratorConfig) error {
	if err := schema.outputSqlTable(table.Table, config); err != nil {
		return err
	}

	for _, aux := range table.AuxTables {
		if err := schema.outputSqlTable(aux, config); err != nil {
			return err
		}
	}

	config.WriteSqlReturnIndent(0)

	return nil
}

func (schema *Schema) outputSqlTable(table *Table, config GeneratorConfig) error {
	if err := config.WriteSqlF("CREATE TABLE %s (", table.Name); err != nil {
		return err
	}
	first := true

	fieldLines := indentMysqlFields(table.SimpleFields)
	for _, field := range fieldLines {
		if !first {
			if err := config.WriteSqlF(","); err != nil {
				return err
			}
		}
		first = false
		if err := config.WriteSqlReturnIndent(1); err != nil {
			return err
		}
		if err := config.WriteSql(field); err != nil {
			return err
		}
	}

	if len(table.PrimaryKeys) > 0 {
		if err := config.WriteSql(","); err != nil {
			return err
		}
		if err := config.WriteSqlReturnIndent(1); err != nil {
			return err
		}
		if err := config.WriteSqlF("PRIMARY KEY (%s)", strings.Join(table.PrimaryKeys, ", ")); err != nil {
			return err
		}
	}
	for indexName, colNames := range table.UniqueKeys {
		if err := config.WriteSql(","); err != nil {
			return err
		}
		if err := config.WriteSqlReturnIndent(1); err != nil {
			return err
		}
		if err := config.WriteSqlF("UNIQUE KEY `%s` (%s)", indexName, strings.Join(colNames, ", ")); err != nil {
			return err
		}
	}
	for indexName, colNames := range table.CompositeKeys {
		if err := config.WriteSql(","); err != nil {
			return err
		}
		if err := config.WriteSqlReturnIndent(1); err != nil {
			return err
		}
		if err := config.WriteSqlF("KEY `%s` (%s)", indexName, strings.Join(colNames, ", ")); err != nil {
			return err
		}
	}
	for _, foreign := range table.ForeignKeys {
		if err := config.WriteSql(","); err != nil {
			return err
		}
		if err := config.WriteSqlReturnIndent(1); err != nil {
			return err
		}
		if err := config.WriteSqlF("FOREIGN KEY (%s) REFERENCES %s(%s) ON UPDATE %s ON DELETE %s",
			strings.Join(foreign.SourceColumns, ", "), foreign.RefTable, strings.Join(foreign.RefColumns, ", "),
			foreign.OnUpdate, foreign.OnDelete,
		); err != nil {
			return err
		}
	}

	if err := config.WriteSqlF("%s);%s", config.Eol, config.Eol); err != nil {
		return err
	}
	return nil
}

func indentMysqlFields(fields []*MysqlField) []string {
	const MysqlFieldLength = 4

	lines := make([][MysqlFieldLength]string, 0, len(fields))
	for _, field := range fields {
		lines = append(lines, [...]string{
			field.Name,
			field.Type,
			elvis.Ternary(field.Nullable, "", "NOT NULL").(string),
			elvis.Ternary(field.AutoIncrement, "AUTO_INCREMENT", "").(string),
		})
	}

	lengths := [MysqlFieldLength]int{}
	for _, line := range lines {
		for i := range lengths {
			if lengths[i] < len(line[i]) {
				lengths[i] = len(line[i])
			}
		}
	}

	ret := make([]string, 0, len(fields))
	for _, line := range lines {
		retLine := ""
		for i, piece := range line {
			retLine += piece
			retLine += strings.Repeat(" ", lengths[i]-len(piece)+1)
		}
		ret = append(ret, strings.TrimRight(retLine, " "))
	}
	return ret
}
