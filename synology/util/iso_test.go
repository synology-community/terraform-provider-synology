package util

import (
	"context"
	"testing"
)

func Test_IsoFromCloudInit(t *testing.T) {
	type args struct {
		ci CloudInit
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test case 1",
			args: args{
				ci: CloudInit{
					MetaData:      "meta-data",
					UserData:      "user-data",
					NetworkConfig: "network-config",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := IsoFromCloudInit(context.Background(), tt.args.ci)
			if (err != nil) != tt.wantErr {
				t.Errorf("isoFromCloudInit() error = %v, wantErr %v", err, tt.wantErr)
				return
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
			removeTmpIsoDirectory(context.Background(), tt.args.iso)
		})
	}
}
