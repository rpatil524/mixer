// Copyright 2024 Google LLC
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

// Model objects related to the spanner graph database.
package spanner

import (
	"encoding/base64"
	"fmt"
	"sort"

	pb "github.com/datacommonsorg/mixer/internal/proto"
	"google.golang.org/protobuf/proto"
)

// Property struct represents a subset of a row in the Edge table.
type Property struct {
	SubjectID string `spanner:"subject_id"`
	Predicate string `spanner:"predicate"`
}

// Edge struct represents a single row in the Edge table, supplemented with the object name and types.
type Edge struct {
	SubjectID   string   `spanner:"subject_id"`
	Predicate   string   `spanner:"predicate"`
	ObjectID    string   `spanner:"object_id"`
	ObjectValue string   `spanner:"object_value"`
	ObjectBytes []byte   `spanner:"object_bytes"`
	Provenance  string   `spanner:"provenance"`
	Name        string   `spanner:"name"`
	Types       []string `spanner:"types"`
}

// Observation struct represents a single row in the Observation table.
type Observation struct {
	VariableMeasured  string     `spanner:"variable_measured"`
	ObservationAbout  string     `spanner:"observation_about"`
	Observations      TimeSeries `spanner:"observations"`
	ImportName        string     `spanner:"import_name"`
	ObservationPeriod string     `spanner:"observation_period"`
	MeasurementMethod string     `spanner:"measurement_method"`
	Unit              string     `spanner:"unit"`
	ScalingFactor     string     `spanner:"scaling_factor"`
	ProvenanceURL     string     `spanner:"provenance_url"`
}

// Single observation in a time series.
// Value is a string to allow series with non-numeric types.
type DateValue struct {
	Date  string
	Value string
}

type TimeSeries []*DateValue

// DecodeSpanner decodes the observations field to a TimeSeries value.
// This is inherited from the spanner Decoder interface to decode from a spanner type to a custom type.
// Reference: https://cloud.google.com/go/docs/reference/cloud.google.com/go/spanner/latest#cloud_google_com_go_spanner_Decoder
// Note that the undecoded value is a base64 encoded string.
func (ts *TimeSeries) DecodeSpanner(val interface{}) (err error) {
	obs := &pb.Observations{}
	decodedVal, err := base64.StdEncoding.DecodeString(val.(string))
	if err != nil {
		return fmt.Errorf("failed to decode base64 encoded string: (%v)", err)
	}
	err = proto.Unmarshal(decodedVal, obs)
	if err != nil {
		return fmt.Errorf("failed to decode Observations: (%v)", err)
	}
	*ts = []*DateValue{}
	for date, value := range obs.Values {
		*ts = append(*ts, &DateValue{
			Date:  date,
			Value: value,
		})
	}
	sort.Slice(*ts, func(i, j int) bool {
		return (*ts)[i].Date < (*ts)[j].Date
	})
	return nil
}

// SearchNode struct represents a single row returned for node searches.
type SearchNode struct {
	SubjectID          string   `spanner:"subject_id"`
	Name               string   `spanner:"name"`
	Types              []string `spanner:"types"`
	MatchedPredicate   string   `spanner:"predicate"`
	MatchedObjectValue string   `spanner:"object_value"`
	Score              float64  `spanner:"score"`
}

// SpannerConfig struct to hold the YAML configuration to a spanner database.
type SpannerConfig struct {
	Project  string `yaml:"project"`
	Instance string `yaml:"instance"`
	Database string `yaml:"database"`
}
