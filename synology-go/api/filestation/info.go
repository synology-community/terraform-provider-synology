package filestation

import (
	"github.com/maksym-nazarenko/terraform-provider-synology/synology-go/api"
)

type FileStationInfoRequest struct {
	baseFileStationRequest
}

type FileStationInfoResponse struct {
	baseFileStationResponse

	IsManager              bool
	SupportVirtualProtocol string
	Supportsharing         bool
	Hostname               string
}

var _ api.Request = (*FileStationInfoRequest)(nil)

func NewFileStationInfoRequest(version int) *FileStationInfoRequest {
	return &FileStationInfoRequest{
		baseFileStationRequest: baseFileStationRequest{
			Version:   version,
			APIName:   "SYNO.FileStation.Info",
			APIMethod: "get",
		},
	}
}

func (r FileStationInfoResponse) ErrorSummaries() []api.ErrorSummary {
	return []api.ErrorSummary{commonErrors}
}
