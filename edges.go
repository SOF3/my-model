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
)

type Edge struct {
	Name      string
	PeerTable string
	Type      EdgeType
}

//go:generate go-stringer-inverse -linecomment -trimprefix=EdgeType -type=EdgeType
type EdgeType uint

const (
	EdgeTypeMultiMulti EdgeType = iota
	EdgeTypeMultiOne
	EdgeTypeMultiOneParent
	EdgeTypeOneMulti
	EdgeTypeOneOne
	EdgeTypeOneOneParent
)

func (schema *Schema) computeEdges() error {
	for _, table := range schema.getSortedTables() {
		edges := make([]*Edge, 0, len(table.Edges)+1)
		for _, edge := range table.Edges {
			edges = append(edges, edge)
		}
		if table.knownParent != nil {
			if table.FindEdgeByPeerTable(table.knownParent.Name) == nil {
				edges = append(edges, &Edge{
					Name:      "_parent_",
					Type:      elvis.Ternary(table.knownParent.FindEdgeByPeerTable(table.Name).Type == EdgeTypeOneMulti, EdgeTypeMultiOneParent, EdgeTypeOneOneParent).(EdgeType),
					PeerTable: table.knownParent.Name,
				})
			}
		}
		for _, edge := range edges {

			peer := schema.mustGetTable(edge.PeerTable)

			switch edge.Type {
			case EdgeTypeMultiMulti:
				aux := NewTable(table.Name + "_" + edge.Name)
				foreign := MakeForeignKey(peer.Name)
				for _, key := range table.PrimaryKeys {
					originalField := *table.FindField(key)
					field := originalField
					field.Name = table.Name + "_" + field.Name
					field.AutoIncrement = false
					foreign.SourceColumns = append(foreign.SourceColumns, field.Name)
					foreign.RefColumns = append(foreign.RefColumns, key)
					aux.SimpleFields = append(aux.SimpleFields, &field)
					aux.PrimaryKeys = append(aux.PrimaryKeys, field.Name)
				}
				for _, key := range peer.PrimaryKeys {
					originalField := *table.FindField(key)
					field := originalField
					field.Name = peer.Name + "_" + field.Name
					field.AutoIncrement = false
					foreign.SourceColumns = append(foreign.SourceColumns, field.Name)
					foreign.RefColumns = append(foreign.RefColumns, key)
					foreign.OnUpdate = ReferenceOptionCascade
					foreign.OnDelete = ReferenceOptionCascade
					aux.SimpleFields = append(aux.SimpleFields, &field)
					aux.PrimaryKeys = append(aux.PrimaryKeys, field.Name)
				}
				table.AuxTables = append(table.AuxTables, aux)

			case EdgeTypeMultiOneParent:
				if edge := peer.FindEdgeByPeerTable(table.Name); edge == nil || edge.Type != EdgeTypeOneMulti {
					return errors.New(fmt.Sprintf("type %[1]s does not contain %[2]s in a non-pointer slice, but is declared as slice parent in %[2]s.%[3]s", peer.Name, table.Name, edge.Name))
				}
				fallthrough
			case EdgeTypeMultiOne:
				keys := peer.PrimaryKeys
				if len(peer.PrimaryKeys) == 0 {
					return errors.New("cannot reference type " + peer.Name + " by pointer in " + table.Name + "." + edge.Name + " because it does not have primary keys")
				}
				foreign := MakeForeignKey(peer.Name)
				for _, key := range keys {
					originalField := *peer.FindField(key) // the definition is in the peer
					field := originalField
					field.Name = peer.Name + "_" + field.Name
					field.AutoIncrement = false
					foreign.SourceColumns = append(foreign.SourceColumns, field.Name)
					foreign.RefColumns = append(foreign.RefColumns, key)
					if edge.Type == EdgeTypeMultiOneParent {
						foreign.OnUpdate = ReferenceOptionCascade
						foreign.OnDelete = ReferenceOptionCascade
					} else {
						foreign.OnUpdate = elvis.Ternary(field.Nullable, ReferenceOptionSetNull, ReferenceOptionRestrict).(ReferenceOption)
						foreign.OnDelete = elvis.Ternary(field.Nullable, ReferenceOptionSetNull, ReferenceOptionRestrict).(ReferenceOption)
					}
					table.SimpleFields = append(table.SimpleFields, &field)
				}
				table.ForeignKeys = append(table.ForeignKeys, foreign)

			case EdgeTypeOneMulti:
				// no need to populate anything here

			case EdgeTypeOneOneParent:
				if edge := peer.FindEdgeByPeerTable(table.Name); edge == nil || edge.Type != EdgeTypeOneOne {
					return errors.New(fmt.Sprintf("type %[1]s does not contain %[2]s directly, but is declared as single parent in %[2]s.%[3]s", peer.Name, table.Name, edge.Name))
				}
				// fallthrough
				// case EdgeTypeOneOne:
				keys := peer.PrimaryKeys
				if len(peer.PrimaryKeys) == 0 {
					return errors.New("cannot reference type " + peer.Name + " by pointer in " + table.Name + "." + edge.Name + " because it does not have primary keys")
				}
				foreign := MakeForeignKey(peer.Name)
				for _, key := range keys {
					var originalField MysqlField
					if edge.Type == EdgeTypeOneOneParent {
						originalField = *peer.FindField(key) // copy the definition from the peer
					} else {
						originalField = *table.FindField(key) // the definition has not been copied yet; just use the local one
					}
					field := originalField
					field.Name = peer.Name + "_" + field.Name
					field.AutoIncrement = false
					foreign.SourceColumns = append(foreign.SourceColumns, field.Name)
					foreign.RefColumns = append(foreign.RefColumns, key)
					if edge.Type == EdgeTypeOneOneParent {
						foreign.OnUpdate = ReferenceOptionCascade
						foreign.OnDelete = ReferenceOptionCascade
					} else {
						foreign.OnUpdate = elvis.Ternary(field.Nullable, ReferenceOptionSetNull, ReferenceOptionRestrict).(ReferenceOption)
						foreign.OnDelete = elvis.Ternary(field.Nullable, ReferenceOptionSetNull, ReferenceOptionRestrict).(ReferenceOption)
					}
					table.SimpleFields = append(table.SimpleFields, &field)
					println("Copied parent primary key", field.Name, "into", table.Name)
				}
				table.ForeignKeys = append(table.ForeignKeys, foreign)
			}
		}
	}

	return nil
}
