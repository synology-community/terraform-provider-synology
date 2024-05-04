package client

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"github.com/appkins/terraform-provider-synology/synology-go/api"
	"github.com/appkins/terraform-provider-synology/synology-go/api/filestation"
	"github.com/appkins/terraform-provider-synology/synology-go/api/virtualization"
	"github.com/appkins/terraform-provider-synology/synology-go/util"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/net/publicsuffix"
)

type Client interface {
	Login(user, password, sessionName string) error
	Do(r api.Request, response api.Response) error
	CreateFolder(folderPath string, name string, forceParent bool) (*filestation.CreateFolderResponse, error)
	ListShares() (*filestation.ListShareResponse, error)
	ListGuests() (*virtualization.ListGuestResponse, error)
}
type client struct {
	httpClient *http.Client
	host       string
}

// ListGuests implements Client.
func (c *client) ListGuests() (*virtualization.ListGuestResponse, error) {
	request := api.NewRequest("SYNO.Virtualization.API.Guest", "list")
	response := virtualization.ListGuestResponse{}
	if err := c.Do(request, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// ListShares implements Client.
func (c *client) ListShares() (*filestation.ListShareResponse, error) {
	panic("unimplemented")
}

type synologyClient struct {
	host    string
	apiInfo map[string]api.InfoData
	sid     string
}

// New initializes "client" instance with minimal input configuration.
func New(host string, skipCertificateVerification bool) (Client, error) {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: skipCertificateVerification,
		},
	}

	// currently, 'Cookie' is the only supported method for providing 'sid' token to DSM
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, err
	}
	httpClient := &http.Client{
		Transport: transport,
		Jar:       jar,
	}

	return &client{
		httpClient: httpClient,
		host:       host,
	}, nil
}

// Login runs a login flow to retrieve session token from Synology.
func (c *client) Login(user, password, sessionName string) error {
	u := c.baseURL()

	u.Path = "/webapi/entry.cgi"
	q := u.Query()
	q.Add("api", "SYNO.API.Auth")
	q.Add("version", "7")
	q.Add("method", "login")
	q.Add("account", user)
	q.Add("passwd", password)
	q.Add("session", sessionName)
	q.Add("format", "cookie")
	u.RawQuery = q.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_, _ = io.ReadAll(resp.Body)
		_ = resp.Body.Close()
	}()

	return nil
}

func (c client) CreateFolder(folderPath string, name string, forceParent bool) (*filestation.CreateFolderResponse, error) {
	request := filestation.NewCreateFolderRequest(2)
	request.WithFolderPath(folderPath)
	request.WithName(name)
	request.WithForceParent(forceParent)

	response := filestation.CreateFolderResponse{}

	err := c.Do(request, &response)

	return &response, err
}

// Do performs an HTTP request to remote Synology instance.
//
// Returns error in case of any transport errors.
// For API-level errors, check response object.
func (c client) Do(r api.Request, response api.Response) error {
	u := c.baseURL()

	// request can override this path by implementing APIPathProvider interface
	u.Path = "/webapi/entry.cgi"
	query, err := util.MarshalURL(r)
	if err != nil {
		return err
	}

	u.RawQuery = query.Encode()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_, _ = io.ReadAll(resp.Body)
		_ = resp.Body.Close()
	}()

	synoResponse := api.GenericResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&synoResponse); err != nil {
		return err
	}
	if err := mapstructure.Decode(synoResponse.Data, response); err != nil {
		return err
	}
	response.SetError(handleErrors(synoResponse, response, api.GlobalErrors))

	return nil
}

func (c client) baseURL() url.URL {
	return url.URL{
		Scheme: "https",
		Host:   c.host,
	}
}

func handleErrors(response api.GenericResponse, errorDescriber api.ErrorDescriber, knownErrors api.ErrorSummary) api.SynologyError {
	err := api.SynologyError{
		Code: response.Error.Code,
	}
	if response.Error.Code == 0 {
		return err
	}

	combinedKnownErrors := append(errorDescriber.ErrorSummaries(), knownErrors)
	err.Summary = api.DescribeError(err.Code, combinedKnownErrors...)
	for _, e := range response.Error.Errors {
		item := api.ErrorItem{
			Code:    e.Code,
			Summary: api.DescribeError(e.Code, combinedKnownErrors...),
		}
		if len(e.Details) > 0 {
			item.Details = make(api.ErrorFields)
			for k, v := range e.Details {
				item.Details[k] = v
			}
		}
		err.Errors = append(err.Errors, item)
	}

	return err
}
