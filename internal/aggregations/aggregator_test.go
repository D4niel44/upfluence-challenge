package aggregations

import (
	"encoding/json"
	"strings"
	"testing"
)

var testPosts = []string{
	"{\"pin\":{\"id\":97636533,\"likes\":0,\"comments\":1,\"saves\":0,\"repins\":0,\"timestamp\":2}}",
	"{\"pin\":{\"id\":97636533,\"likes\":0,\"comments\":10,\"saves\":0,\"repins\":0,\"timestamp\":100}}",
	"{\"pin\":{\"id\":97636533,\"likes\":0,\"saves\":0,\"repins\":0,\"timestamp\":10}}",
	"{}",
	"{\"tweet\":{\"id\":97636533,\"likes\":0,\"comments\":5,\"saves\":0,\"repins\":0,\"timestamp\":1}}",
	"{\"pin\":{\"id\":97636533,\"likes\":0,\"comments\":0,\"saves\":0,\"repins\":0,\"timestamp\":1}}",
}

func encodePosts() (posts []map[string]interface{}) {
	for _, post := range testPosts {
		d := json.NewDecoder(strings.NewReader(post))
		d.UseNumber()
		m := make(map[string]interface{})
		d.Decode(&m)
		posts = append(posts, m)
	}
	return
}

func assertEqual[T comparable](t *testing.T, expected T, actual T) {
	if actual != expected {
		t.Error("Expected", expected, "but got", actual)
	}
}

func TestAggregate(t *testing.T) {
	posts := encodePosts()

	state := InitialState("comments")
	for _, post := range posts {
		err := state.Aggregate(post)
		if err != nil {
			t.Error(err)
		}
	}
	assertEqual(t, "comments", state.Dimension)
	assertEqual(t, 4, state.DimensionAvg)
	assertEqual(t, 1, state.MinTimestamp)
	assertEqual(t, 100, state.MaxTimestamp)
	assertEqual(t, 5, state.TotalPosts)
}
