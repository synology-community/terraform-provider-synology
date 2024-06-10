package util

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestGetType(t *testing.T) {
	type args struct {
		r interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]attr.Type
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
			want: map[string]attr.Type{
				"name":   types.StringType,
				"age":    types.NumberType,
				"states": types.ListType{}.WithElementType(types.StringType),
				"addresses": types.ListType{}.WithElementType(types.ObjectType{}.WithAttributeTypes(map[string]attr.Type{
					"street": types.StringType,
					"city":   types.StringType,
				},
				)),
			},
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
