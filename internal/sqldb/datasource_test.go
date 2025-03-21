// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sqldb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestId(t *testing.T) {
	sqlClient, err := NewSQLiteClient("../../test/sqlquery/key_value/datacommons.db")
	if err != nil {
		t.Fatalf("Could not open test database: %v", err)
	}

	assert.Equal(t, "sqlite-../../test/sqlquery/key_value/datacommons.db", sqlClient.id)

	sqlDataSource := NewSQLDataSource(sqlClient, nil)
	assert.Equal(t, "sql-sqlite-../../test/sqlquery/key_value/datacommons.db", sqlDataSource.Id())
}
