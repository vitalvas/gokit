package xstrings

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestReplaceMap(t *testing.T) {
	data := map[string]string{
		"aa":    "there",
		"bb":    "no",
		"zz":    "yes",
		"empty": "full",
	}

	payload := "hello, aa, bb, zz"
	correctPayload := "hello, there, no, yes"

	replaced := ReplaceMap(payload, data)

	assert.Equal(t, correctPayload, replaced)
}
