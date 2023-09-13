package main

import (
	"testing"

	"github.com/go-playground/assert/v2"
)

func Test_SpecialStringtoArray(t *testing.T) {

	testcases := []struct {
		what   string
		input  string
		output []string
	}{{
		what:   "single element",
		input:  "0-3",
		output: []string{"0", "1", "2", "3"},
	}, {
		what:   "double element",
		input:  "0-1,3-4",
		output: []string{"0", "1", "3", "4"},
	}, {
		what:   "triple element 1",
		input:  "0-1,3-4,95",
		output: []string{"0", "1", "3", "4", "95"},
	}, {
		what:   "triple element 2",
		input:  "0-1,3-4,95-99",
		output: []string{"0", "1", "3", "4", "95", "96", "97", "98", "99"},
	}, {
		what:   "triple element with whitespace in the beginning",
		input:  "  0-1,3-4,95-99",
		output: []string{"0", "1", "3", "4", "95", "96", "97", "98", "99"},
	}, {
		what:   "triple element with whitespace at the end",
		input:  "0-1,3-4,95-99  ",
		output: []string{"0", "1", "3", "4", "95", "96", "97", "98", "99"},
	},
	}

	for _, tc := range testcases {
		t.Run(tc.what, func(t *testing.T) {
			actual := SpecialStringtoArray(tc.input)
			assert.Equal(t, tc.output, actual)
		})
	}
}
