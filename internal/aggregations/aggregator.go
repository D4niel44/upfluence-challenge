// Package aggregations implements aggregations of values for events from https://stream.upfluence.co/stream.
package aggregations

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
)

// AggregatorState stores the current
// state of the aggregations, and can be to aggregate new values
type AggregatorState struct {
	TotalPosts     int64
	MinTimestamp   int64
	MaxTimestamp   int64
	Dimension      string
	DimensionAvg   float64
	DimensionCount int64
}

// InitialState returns the initial state of the aggregator.
func InitialState(dimension string) *AggregatorState {
	return &AggregatorState{
		TotalPosts:     0,
		MinTimestamp:   math.MaxInt,
		MaxTimestamp:   math.MinInt,
		Dimension:      dimension,
		DimensionAvg:   0,
		DimensionCount: 0,
	}
}

// Aggregate aggregates a new post to the current AggregatorState, modifying the state in place.
//
// An empty map is not considered a post, calling this method with an empty map will not perform
// any modification to the AggregatorState.
//
// A post without the required dimension is not considered an error, the dimension average is not
// modified, but the other metrics (min and max timestamp, total count) are updated.
func (state *AggregatorState) Aggregate(post map[string]interface{}) error {
	if len(post) == 0 { // Events of type data: {} are not considered posts.
		return nil
	}
	if len(post) > 1 {
		return aggregationError("Could not parse post")
	}
	for _, v := range post {
		switch postData := v.(type) {
		case map[string]interface{}:
			state.TotalPosts++

			timestampJSON, ok := postData["timestamp"].(json.Number)
			if !ok {
				return aggregationError("Could not read timestamp")
			}
			timestamp, err := timestampJSON.Int64()
			if err != nil {
				return aggregationError("Could not parse timestamp")
			}

			if timestamp < state.MinTimestamp {
				state.MinTimestamp = timestamp
			}
			if timestamp > state.MaxTimestamp {
				state.MaxTimestamp = timestamp
			}

			dimensionJSON, ok := postData[state.Dimension].(json.Number)
			if !ok {
				if postData[state.Dimension] == nil {
					return nil // A post may not have a given dimension.
				}
				return aggregationError("Could not read dimension")
			}
			dimensionValue, err := dimensionJSON.Float64()
			if err != nil {
				return aggregationError("Could not parse dimension")
			}
			state.DimensionCount++
			state.DimensionAvg = state.DimensionAvg + (dimensionValue-state.DimensionAvg)/float64(state.DimensionCount)
		default:
			return aggregationError("Could not parse post")
		}
	}
	return nil
}

func aggregationError(msg ...any) error {
	return errors.New(fmt.Sprint("Error aggregating post, ", msg))
}
