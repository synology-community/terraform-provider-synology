package api

// Response defines an interface for all responses from Synology API.
type Response interface {
	ErrorDescriber

	// GetError returns the latest error associated with response, if any.
	GetError() SynologyError

	// SetError sets error object for the current response.
	SetError(SynologyError)

	// Success reports whether the current request was successful.
	Success() bool
}

// GenericResponse is a concrete Response implementation.
// It is a generic struct with common to all Synology response fields.
type GenericResponse struct {
	Success bool
	Data    interface{}
	Error   SynologyError
}

type BaseResponse struct {
	SynologyError
}

func (b *BaseResponse) SetError(e SynologyError) {
	b.SynologyError = e
}

func (b BaseResponse) Success() bool {
	return b.SynologyError.Code == 0
}

func (b *BaseResponse) GetError() SynologyError {
	return b.SynologyError
}
