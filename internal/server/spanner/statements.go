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

// Query statements used by the SpannerClient.
package spanner

import (
	"fmt"

	"github.com/datacommonsorg/mixer/internal/merger"
)

// SQL / GQL statements executed by the SpannerClient
var statements = struct {
	// Fetch Properties for out arcs.
	getPropsBySubjectID string
	// Fetch Properties for in arcs.
	getPropsByObjectID string
	// Fetch Edges for out arcs with a single hop.
	getEdgesBySubjectID string
	// Fetch Edges for out arcs with chaining.
	getChainedEdgesBySubjectID string
	// Fetch Edges for in arcs with a single hop.
	getEdgesByObjectID string
	// Fetch Edges for in arcs with chaining.
	getChainedEdgesByObjectID string
	// Subquery to filter edges by predicate.
	filterProps string
	// Subquery to filter edges by object property-values.
	filterObjects string
	// Subquery to apply page offset.
	applyOffset string
	// Subquery to apply page limit.
	applyLimit string
	// Fetch Observations.
	getObs string
	// Filter by variable dcids.
	selectVariableDcids string
	// Filter by entity dcids.
	selectEntityDcids string
	// Fetch observations for variable + contained in place.
	getObsByVariableAndContainedInPlace string
	// Search nodes by name only.
	searchNodesByQuery string
	// Search nodes by query and type(s).
	searchNodesByQueryAndTypes string
	// Search object values by query, predicate and types.
	searchObjectValues string
	// Subquery to filter search results by types.
	filterTypes string
}{
	getPropsBySubjectID: `
		GRAPH DCGraph MATCH -[e:Edge
		WHERE 
			e.subject_id IN UNNEST(@ids)]->
		RETURN DISTINCT
			e.subject_id,
			e.predicate
		ORDER BY
			e.subject_id,
			e.predicate
	`,
	getPropsByObjectID: `
		GRAPH DCGraph MATCH -[e:Edge
		WHERE
			e.object_id IN UNNEST(@ids)
			AND e.subject_id != e.object_id
		]->
		RETURN DISTINCT
			e.object_id AS subject_id,
			e.predicate
		ORDER BY
			subject_id,
			predicate
	`,
	getEdgesBySubjectID: `
		GRAPH DCGraph MATCH -[e:Edge]->(n:Node)
		WHERE
			e.subject_id IN UNNEST(@ids)
			AND e.subject_id != e.object_id%[1]s
		RETURN 
			e.subject_id,
			e.predicate,
			e.object_id,
			e.object_value,
			e.object_bytes,
			e.provenance,
			n.name,
			n.types
		UNION ALL
		MATCH -[e:Edge]->
		WHERE
			e.subject_id IN UNNEST(@ids)
			AND e.subject_id = e.object_id%[1]s
		RETURN 
			e.subject_id,
			e.predicate,
			'' as object_id,
			e.object_value,
			e.object_bytes,
			e.provenance,
			'' AS name,
			ARRAY<STRING>[] AS types
		NEXT
		RETURN
			subject_id,
			predicate,
			object_id,
			COALESCE(object_value, '') AS object_value,
			object_bytes,
			provenance,
			name,
			types
		ORDER BY
			subject_id,
			predicate,
			object_id,
			object_value,
			object_bytes,
			provenance
	`,
	getChainedEdgesBySubjectID: `
		GRAPH DCGraph MATCH ANY (m:Node)-[e:Edge
		WHERE
			e.predicate = @predicate]->{1,%d}(n:Node)
		WHERE
			m.subject_id IN UNNEST(@ids)
			AND m != n
		RETURN 
			m.subject_id,
			n.subject_id AS object_id,
			'' AS object_value,
			CAST(NULL AS BYTES) AS object_bytes,
			COALESCE(n.name, '') AS name,
			COALESCE(n.types, []) AS types
		UNION ALL
		MATCH -[e:Edge]->
		WHERE
			e.subject_id IN UNNEST(@ids)
			AND e.subject_id = e.object_id
			AND e.predicate = @predicate
		RETURN 
			e.subject_id,
			'' AS object_id,
			COALESCE(e.object_value, '') AS object_value,
			e.object_bytes,
			'' AS name,
			ARRAY<STRING>[] AS types
		NEXT
		RETURN
			subject_id,
			@result_predicate AS predicate,
			object_id,
			object_value,
			object_bytes,
			'' AS provenance,
			name, 
			types
		ORDER BY
			subject_id,
			predicate,
			object_id,
			object_value,
			object_bytes
	`,
	getEdgesByObjectID: `
		GRAPH DCGraph MATCH <-[e:Edge]-(n:Node) 
		WHERE
			e.object_id IN UNNEST(@ids)
			AND e.subject_id != e.object_id%s
		RETURN 
			e.object_id AS subject_id,
			e.predicate,
			e.subject_id AS object_id,
			'' AS object_value,
			e.object_bytes,
			COALESCE(e.provenance, '') AS provenance,
			COALESCE(n.name, '') AS name,
			COALESCE(n.types, []) AS types
		ORDER BY
			subject_id,
			predicate,
			object_id
	`,
	getChainedEdgesByObjectID: `
		GRAPH DCGraph MATCH ANY (m:Node)<-[e:Edge
		WHERE
			e.predicate = @predicate]-{1,%d}(n:Node) 
		WHERE 
			m.subject_id IN UNNEST(@ids)
			AND m!= n
		RETURN 
			m.subject_id,
			@result_predicate AS predicate,
			n.subject_id AS object_id,
			'' AS object_value,
			'' AS provenance, 
			CAST(NULL AS BYTES) AS object_bytes,
			COALESCE(n.name, '') AS name,
			COALESCE(n.types, []) AS types
		ORDER BY
			subject_id,
			predicate,
			object_id
		`,
	filterProps: `
		AND e.predicate IN UNNEST(@props)
	`,
	filterObjects: `
		NEXT 
		MATCH -[filter:Edge 
		WHERE
			filter.predicate = @prop%[1]d
			AND (
				filter.object_id IN UNNEST(@val%[1]d)
				OR filter.object_value IN UNNEST(@val%[1]d)
			)]-> 
		WHERE
			filter.subject_id = object_id
		RETURN
			subject_id,
			predicate,
			object_id,
			object_value,
			object_bytes,
			provenance,
			name,
			types			
	`,
	applyOffset: `
		OFFSET %d
	`,
	applyLimit: fmt.Sprintf(`
		LIMIT %d
	`, PAGE_SIZE+1),
	getObs: `
		SELECT
			variable_measured,
			observation_about,
			observations,
			import_name,
			COALESCE(observation_period, '') AS observation_period,
			COALESCE(measurement_method, '') AS measurement_method,
			COALESCE(unit, '') AS unit,
			COALESCE(scaling_factor, '') AS scaling_factor,
			provenance_url
		FROM 
			Observation
	`,
	selectVariableDcids: `
		variable_measured IN UNNEST(@variables)
	`,
	selectEntityDcids: `
		observation_about IN UNNEST(@entities)
	`,
	getObsByVariableAndContainedInPlace: `
		SELECT
			obs.variable_measured,
			obs.observation_about,
			obs.observations,
			obs.import_name,
			obs.observation_period,
			obs.measurement_method,
			obs.unit,
			obs.scaling_factor,
			obs.provenance_url
		FROM 
			GRAPH_TABLE (
				DCGraph MATCH <-[e:Edge
				WHERE
					e.object_id = @ancestor
					AND e.subject_id != e.object_id
					AND e.predicate = 'linkedContainedInPlace']-()-[{predicate: 'typeOf', object_id: @childPlaceType}]->
				RETURN 
				e.subject_id as object_id
			)result
		INNER JOIN (%s)obs
		ON 
			result.object_id = obs.observation_about
	`,
	searchNodesByQuery: fmt.Sprintf(`
		GRAPH DCGraph
		MATCH (n:Node)
		WHERE 
			SEARCH(n.name_tokenlist, @query)
		RETURN 
			n.subject_id, 
			COALESCE(n.name, '') AS name,
			COALESCE(n.types, []) AS types, 
			SCORE(n.name_tokenlist, @query, enhance_query => TRUE) AS score 
		ORDER BY score + IF(n.name = @query, 1, 0) DESC, n.name ASC
		LIMIT %d
	`, merger.MAX_SEARCH_RESULTS),
	searchNodesByQueryAndTypes: fmt.Sprintf(`
		GRAPH DCGraph
		MATCH (n:Node)
		WHERE 
			SEARCH(n.name_tokenlist, @query)
			AND ARRAY_INCLUDES_ANY(n.types, @types)
		RETURN 
			n.subject_id, 
			COALESCE(n.name, '') AS name, 
			COALESCE(n.types, []) AS types, 
			SCORE(n.name_tokenlist, @query, enhance_query => TRUE) AS score
		ORDER BY score + IF(n.name = @query, 1, 0) DESC, n.name ASC
		LIMIT %d
	`, merger.MAX_SEARCH_RESULTS),
	searchObjectValues: `
		GRAPH DCGraph 
		MATCH -[e:Edge 
			WHERE e.predicate IN UNNEST(@predicates) AND SEARCH(e.object_value_tokenlist, @query)
		]->(n:Node %s)
		RETURN 
			n.subject_id, 
			COALESCE(n.name, '') AS name, 
			COALESCE(n.types, []) AS types, 
			e.predicate AS predicate, 
			e.object_value AS object_value, 
			SCORE(e.object_value_tokenlist, @query, enhance_query => TRUE) AS score
		ORDER BY score + IF(e.object_value = @query, 1, 0) + IF(REGEXP_CONTAINS(n.subject_id, @query), 0.5, 0) DESC, n.name ASC
		LIMIT %d
	`,
	filterTypes: `WHERE ARRAY_INCLUDES_ANY(n.types, @types)`,
}
