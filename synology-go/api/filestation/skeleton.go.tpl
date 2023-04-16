package filestation

import (
	"github.com/maksym-nazarenko/terraform-provider-synology/synology-go/client/api"
)

type __TEMPLATE_TYPE_PLACEHOLDER__Request struct {
	baseFileStationRequest
}

type __TEMPLATE_TYPE_PLACEHOLDER__Response struct {
}

var _ api.Request = (*__TEMPLATE_TYPE_PLACEHOLDER__Request)(nil)

func New__TEMPLATE_TYPE_PLACEHOLDER__Request(version int) *__TEMPLATE_TYPE_PLACEHOLDER__Request {
	return &__TEMPLATE_TYPE_PLACEHOLDER__Request{
		baseFileStationRequest: baseFileStationRequest{
			Version:   version,
			APIName:   "SYNO.FileStation.__TEMPLATE_TYPE_PLACEHOLDER__",
			APIMethod: "_",
		},
	}
}

func (r __TEMPLATE_TYPE_PLACEHOLDER__Rsponse) ErrorSummaries() []api.ErrorSummary {
	return []api.ErrorSummary{
		{
		},
		commonErrors,
	}
}
