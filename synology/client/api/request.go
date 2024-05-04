package api

// Request defines a contract for all Request implementations.
type Request interface{}

type BaseRequest struct {
	Version   int    `synology:"version"`
	APIName   string `synology:"api"`
	APIMethod string `synology:"method"`
}

func NewRequest(apiName, apiMethod string) *BaseRequest {
	return &BaseRequest{
		Version:   ApiVersions[apiName],
		APIName:   apiName,
		APIMethod: apiMethod,
	}
}
