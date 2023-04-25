package client

import (
	"net/url"
	"os"
	"testing"

	"github.com/maksym-nazarenko/terraform-provider-synology/synology-go/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newClient() (Client, error) {
	c, err := New("dev-synology:5001", true)
	if err != nil {
		return nil, err
	}

	if err := c.Login("api-client", os.Getenv("SYNOLOGY_PASSWORD"), "webui"); err != nil {
		return nil, err
	}

	return c, nil
}

func TestMarshalURL(t *testing.T) {
	type embeddedStruct struct {
		EmbeddedString string `synology:"embedded_string"`
		EmbeddedInt    int    `synology:"embedded_int"`
	}

	testCases := []struct {
		name     string
		in       interface{}
		expected url.Values
	}{
		{
			name: "scalar types",
			in: struct {
				Name    string `synology:"name"`
				ID      int    `synology:"id"`
				Enabled bool   `synology:"enabled"`
			}{
				Name:    "name value",
				ID:      2,
				Enabled: true,
			},
			expected: url.Values{
				"name":    []string{"name value"},
				"id":      []string{"2"},
				"enabled": []string{"true"},
			},
		},
		{
			name: "slice types",
			in: struct {
				Names []string `synology:"names"`
				IDs   []int    `synology:"ids"`
			}{
				Names: []string{"value 1", "value 2"},
				IDs:   []int{1, 2, 3},
			},
			expected: url.Values{
				"names": []string{"[\"value 1\",\"value 2\"]"},
				"ids":   []string{"[1,2,3]"},
			},
		},
		{
			name: "embedded struct",
			in: struct {
				embeddedStruct
				Name string `synology:"name"`
			}{
				embeddedStruct: embeddedStruct{
					EmbeddedString: "my string",
					EmbeddedInt:    5,
				},
				Name: "field name",
			},
			expected: url.Values{
				"name":            []string{"field name"},
				"embedded_string": []string{"my string"},
				"embedded_int":    []string{"5"},
			},
		},
		{
			name: "unexported field without tag",
			in: struct {
				Name       string `synology:"name"`
				ID         int    `synology:"id"`
				unexported string
			}{
				Name:       "name value",
				ID:         2,
				unexported: "must be skipped",
			},
			expected: url.Values{
				"name": []string{"name value"},
				"id":   []string{"2"},
			},
		},
		{
			name: "unexported field with tag",
			in: struct {
				Name       string `synology:"name"`
				ID         int    `synology:"id"`
				unexported string `synology:"unexported"`
			}{
				Name:       "name value",
				ID:         2,
				unexported: "with explicit tag",
			},
			expected: url.Values{
				"name":       []string{"name value"},
				"id":         []string{"2"},
				"unexported": []string{"with explicit tag"},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := marshalURL(tc.in)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestHandleErrors(t *testing.T) {
	globalErrors := api.ErrorSummary{
		100: "global error 100",
		101: "global error 101",
		102: "global error 102",
	}

	testCases := []struct {
		name                string
		response            api.GenericResponse
		responseKnownErrors []api.ErrorSummary
		expected            api.SynologyError
	}{
		{
			name: "global errors only",
			response: api.GenericResponse{
				Success: false,
				Data:    nil,
				Error: api.SynologyError{
					Code: 100,
					Errors: []api.ErrorItem{
						{Code: 101},
						{Code: 102, Details: api.ErrorFields{"path": "/some/path", "code": 102, "reason": "a reason"}},
					},
				},
			},
			expected: api.SynologyError{
				Code:    100,
				Summary: "global error 100",
				Errors: []api.ErrorItem{
					{
						Code:    101,
						Summary: "global error 101",
					},
					{
						Code:    102,
						Summary: "global error 102",
						Details: api.ErrorFields{"path": "/some/path", "code": 102, "reason": "a reason"},
					},
				},
			},
		},
		{
			name: "response-specific error",
			response: api.GenericResponse{
				Success: false,
				Data:    nil,
				Error: api.SynologyError{
					Code: 100,
					Errors: []api.ErrorItem{
						{Code: 101},
						{Code: 202, Details: api.ErrorFields{"code": 202}},
					},
				},
			},
			responseKnownErrors: []api.ErrorSummary{
				{
					202: "response error 202",
				},
			},
			expected: api.SynologyError{
				Code:    100,
				Summary: "global error 100",
				Errors: []api.ErrorItem{
					{
						Code:    101,
						Summary: "global error 101",
					},
					{
						Code:    202,
						Summary: "response error 202",
						Details: api.ErrorFields{"code": 202},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := handleErrors(tc.response,
				errorDescriber(func() []api.ErrorSummary { return tc.responseKnownErrors }),
				globalErrors,
			)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

type errorDescriber func() []api.ErrorSummary

func (d errorDescriber) ErrorSummaries() []api.ErrorSummary {
	return d()
}
