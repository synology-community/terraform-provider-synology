// Package api provides types for common objects required during calls to remote Synology instance.
package api

// Request defines a contract for all Request implementations.
type Request interface{}

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
