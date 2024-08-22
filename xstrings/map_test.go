package xstrings

import (
	"reflect"
	"testing"
)

func TestSortMap(t *testing.T) {
	data := map[string]string{
		"zz": "yes",
		"aa": "there",
		"bb": "no",
	}
	correctData := map[string]string{
		"aa": "there",
		"bb": "no",
		"zz": "yes",
	}

	sorted := SortMap(data)

	if !reflect.DeepEqual(sorted, correctData) {
		t.Error("wrong sort map")
	}
}
