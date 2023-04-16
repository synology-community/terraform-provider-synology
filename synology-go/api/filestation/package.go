package filestation

import "github.com/maksym-nazarenko/terraform-provider-synology/synology-go/api"

type baseFileStationRequest struct {
	Version   int    `synology:"version"`
	APIName   string `synology:"api"`
	APIMethod string `synology:"method"`
}

type baseFileStationResponse struct {
	synologyError api.SynologyError
}

func (b *baseFileStationResponse) SetError(e api.SynologyError) {
	b.synologyError = e
}

func (b baseFileStationResponse) Success() bool {
	return b.synologyError.Code == 0
}

func (b *baseFileStationResponse) GetError() api.SynologyError {
	return b.synologyError
}
