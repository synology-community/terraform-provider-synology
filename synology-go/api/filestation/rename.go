package filestation

import (
	"github.com/maksym-nazarenko/terraform-provider-synology/synology-go/api"
)

type FileStationRenameRequest struct {
	baseFileStationRequest

	name string
	path string
}

type File struct {
	Path  string
	Name  string
	IsDir bool
}

type FileStationRenameResponse struct {
	baseFileStationResponse

	Files []File
}

var _ api.Request = (*FileStationRenameRequest)(nil)

func NewFileStationRenameRequest(version int) *FileStationRenameRequest {
	return &FileStationRenameRequest{
		baseFileStationRequest: baseFileStationRequest{
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
