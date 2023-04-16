package client

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/maksym-nazarenko/terraform-provider-synology/synology-go/api"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/net/publicsuffix"
)

type client struct {
	httpClient *http.Client
	host       string
}

// New initializes "client" instance with minimal input configuration.
func New(host string, skipCertificateVerification bool) (*client, error) {
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

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	// req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
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

// Do performs an HTTP request to remote Synology instance.
//
// Returns error in case of any transport errors.
// For API-level errors, check response object.
func (c client) Do(r api.Request, response api.Response) error {
	u := c.baseURL()

	// request can override this path by implementing APIPathProvider interface
	u.Path = "/webapi/entry.cgi"
	query, err := marshalURL(r)
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

func marshalURL(r interface{}) (url.Values, error) {
	v := reflect.Indirect(reflect.ValueOf(r))
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected type struct, got %T", reflect.TypeOf(r).Name())
	}
	n := v.NumField()
	vT := v.Type()
	ret := url.Values{}
	for i := 0; i < n; i++ {
		urlFieldName := strings.ToLower(vT.Field(i).Name)
		synologyTags := []string{}
		if tags, ok := vT.Field(i).Tag.Lookup("synology"); ok {
			synologyTags = strings.Split(tags, ",")
		}
		if !(vT.Field(i).IsExported() || vT.Field(i).Anonymous || len(synologyTags) > 0) {
			continue
		}
		if len(synologyTags) > 0 {
			urlFieldName = synologyTags[0]
		}

		// get field type
		switch vT.Field(i).Type.Kind() {
		case reflect.String:
			ret.Add(urlFieldName, v.Field(i).String())
		case reflect.Int:
			ret.Add(urlFieldName, strconv.Itoa(int(v.Field(i).Int())))
		case reflect.Bool:
			ret.Add(urlFieldName, strconv.FormatBool(v.Field(i).Bool()))
		case reflect.Slice:
			slice := v.Field(i)
			switch vT.Field(i).Type.Elem().Kind() {
			case reflect.String:
				res := []string{}
				for iSlice := 0; iSlice < slice.Len(); iSlice++ {
					item := slice.Index(iSlice)
					res = append(res, item.String())
				}
				ret.Add(urlFieldName, "[\""+strings.Join(res, "\",\"")+"\"]")
			case reflect.Int:
				res := []string{}
				for iSlice := 0; iSlice < slice.Len(); iSlice++ {
					item := slice.Index(iSlice)
					res = append(res, strconv.Itoa(int(item.Int())))
				}
				ret.Add(urlFieldName, "["+strings.Join(res, ",")+"]")
			}
		case reflect.Struct:
			if !vT.Field(i).Anonymous {
				// support only embedded anonymous structs
				continue
			}
			embStruct := v.Field(i)
			embStructT := v.Field(i).Type()
			for j := 0; j < embStruct.NumField(); j++ {
				synologyTags := strings.Split(embStructT.Field(j).Tag.Get("synology"), ",")
				fieldName := synologyTags[0]
				switch embStruct.Field(j).Kind() {
				case reflect.String:
					ret.Add(fieldName, embStruct.Field(j).String())
				case reflect.Int:
					ret.Add(fieldName, strconv.Itoa(int(embStruct.Field(j).Int())))
				}
			}
		}
	}

	return ret, nil
}
