package util

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	addressAttrType = map[string]attr.Type{
		"street": types.StringType,
		"city":   types.StringType,
	}

	testType = map[string]attr.Type{
		"name":   types.StringType,
		"age":    types.NumberType,
		"states": types.ListType{}.WithElementType(types.StringType),
		"addresses": types.ListType{}.WithElementType(
			types.ObjectType{}.WithAttributeTypes(addressAttrType),
		),
	}

	testValues = map[string]attr.Value{
		"name": types.StringValue("John"),
		"age":  types.NumberValue(new(big.Float).SetInt64(30)),
		"states": types.ListValueMust(
			types.StringType,
			[]attr.Value{types.StringValue("CA"), types.StringValue("NY")},
		),
		"addresses": types.ListValueMust(
			types.ObjectType{}.WithAttributeTypes(addressAttrType),
			[]attr.Value{
				types.ObjectValueMust(addressAttrType, map[string]attr.Value{
					"street": types.StringValue("123 Main St"),
					"city":   types.StringValue("San Francisco"),
				}),
			}),
	}
)

func TestGetType(t *testing.T) {
	type args struct {
		r any
	}
	tests := []struct {
		name    string
		args    args
		want    attr.Type
		wantErr bool
	}{
		{
			name: "TestGetType",
			args: args{
				r: struct {
					Name      string   `json:"name"`
					Age       int      `json:"age"`
					States    []string `json:"states"`
					Addresses []struct {
						Street string `json:"street"`
						City   string `json:"city"`
					} `json:"addresses"`
				}{},
			},
			want: types.ObjectType{}.WithAttributeTypes(map[string]attr.Type{
				"name":   types.StringType,
				"age":    types.NumberType,
				"states": types.ListType{}.WithElementType(types.StringType),
				"addresses": types.ListType{}.WithElementType(
					types.ObjectType{}.WithAttributeTypes(map[string]attr.Type{
						"street": types.StringType,
						"city":   types.StringType,
					},
					),
				),
			}),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetType(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetValue(t *testing.T) {
	type args struct {
		r any
	}
	tests := []struct {
		name    string
		args    args
		want    attr.Value
		wantErr bool
	}{
		{
			name: "TestGetValue",
			args: args{
				r: struct {
					Name      string   `json:"name"`
					Age       int      `json:"age"`
					States    []string `json:"states"`
					Addresses []struct {
						Street string `json:"street"`
						City   string `json:"city"`
					} `json:"addresses"`
				}{
					Name:   "John",
					Age:    30,
					States: []string{"CA", "NY"},
					Addresses: []struct {
						Street string `json:"street"`
						City   string `json:"city"`
					}{
						{
							Street: "123 Main St",
							City:   "San Francisco",
						},
					},
				},
			},
			want:    types.ObjectValueMust(testType, testValues),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetValue(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetType() = %v, want %v", got, tt.want)
			}
		})
	}
}
