package filestation

import (
	"github.com/appkins/terraform-provider-synology/synology-go/api"
)

type FileStationInfoRequest struct {
	api.BaseRequest
}

type FileStationInfoResponse struct {
	api.BaseResponse

	IsManager              bool
	SupportVirtualProtocol string
	Supportsharing         bool
	Hostname               string
}

var _ api.Request = (*FileStationInfoRequest)(nil)

func NewFileStationInfoRequest(version int) *FileStationInfoRequest {
	return &FileStationInfoRequest{
		BaseRequest: api.BaseRequest{
			Version:   version,
			APIName:   "SYNO.FileStation.Info",
			APIMethod: "get",
		},
	}
}

func (r FileStationInfoResponse) ErrorSummaries() []api.ErrorSummary {
	return []api.ErrorSummary{commonErrors}
}
