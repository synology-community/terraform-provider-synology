package api

type ApiInfoResponse struct {
	BaseResponse
	ApiInfo
}

type ApiInfo = map[string]InfoData

type InfoData struct {
	Path          string
	MinVersion    int
	MaxVersion    int
	RequestFormat string
}
