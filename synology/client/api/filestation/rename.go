package filestation

import (
	"github.com/appkins/terraform-provider-synology/synology/client/api"
)

type FileStationRenameRequest struct {
	api.BaseRequest

	name string
	path string
}

type File struct {
	Path  string
	Name  string
	IsDir bool
}

type FileStationRenameResponse struct {
	api.BaseResponse

	Files []File
}

var _ api.Request = (*FileStationRenameRequest)(nil)

func NewFileStationRenameRequest(version int) *FileStationRenameRequest {
	return &FileStationRenameRequest{
		BaseRequest: api.BaseRequest{
			Version:   version,
			APIName:   "SYNO.FileStation.Rename",
			APIMethod: "rename",
		},
	}
}

func (r *FileStationRenameRequest) WithName(value string) *FileStationRenameRequest {
	r.name = value
	return r
}

func (r *FileStationRenameRequest) WithPath(value string) *FileStationRenameRequest {
	r.path = value
	return r
}

func (r FileStationRenameResponse) ErrorSummaries() []api.ErrorSummary {
	return []api.ErrorSummary{
		{
			1200: "Failed to rename it.",
		},
		commonErrors,
	}
}
