package util

import (
	"os"
	"reflect"
	"testing"
)

func Test_IsoFromCloudInit(t *testing.T) {
	type args struct {
		ci CloudInit
	}
	tests := []struct {
		name    string
		args    args
		want    *os.File
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IsoFromCloudInit(tt.args.ci)
			if (err != nil) != tt.wantErr {
				t.Errorf("isoFromCloudInit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("isoFromCloudInit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_removeTmpIsoDirectory(t *testing.T) {
	type args struct {
		iso string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			removeTmpIsoDirectory(tt.args.iso)
		})
	}
}
