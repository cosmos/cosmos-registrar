package utils

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToFromJSON(t *testing.T) {

	type stuff struct {
		Address           string
		IsSeed            bool
		ID                string
		LastContactHeight int64
		Reachable         bool
	}

	tests := []struct {
		path    string
		dataOut interface{}
		dataIn  []stuff
	}{
		{
			path: "peers.json",
			dataOut: []stuff{
				{
					Address:           "localhost",
					IsSeed:            true,
					ID:                "xyz",
					LastContactHeight: 134143,
					Reachable:         true,
				},
			},
			dataIn: []stuff{},
		},
	}
	for i, tt := range tests {
		name := fmt.Sprint("TestToFromJSON#", i)
		t.Run(name, func(t *testing.T) {
			p, _ := os.CreateTemp("", "to_from_json")
			assert.True(t, PathExists(p.Name()))
			t.Log("file is", p)
			// Write data
			err := ToJSON(p.Name(), tt.dataOut)
			if err != nil {
				t.Errorf("TestToFromJSON() error = %v", err)
			}
			// Read Data
			err = FromJSON(p.Name(), &tt.dataIn)
			if err != nil {
				t.Errorf("TestToFromJSON() error = %v", err)
			}
			assert.Equal(t, tt.dataOut, tt.dataIn)
		})
	}
}

func TestContainsStr(t *testing.T) {
	tests := []struct {
		elements *[]string
		needle   string
		want     bool
	}{
		{
			&[]string{"A", "b", "c"},
			"a",
			false,
		},
		{
			&[]string{"Alice", "bob", "mark"},
			"bob",
			true,
		},
	}
	for i, tt := range tests {
		name := fmt.Sprint("TestContainsStr#", i)
		t.Run(name, func(t *testing.T) {
			if got := ContainsStr(tt.elements, tt.needle); got != tt.want {
				t.Errorf("ContainsStr() = %v, want %v", got, tt.want)
			}
		})
	}
}
