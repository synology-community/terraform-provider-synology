// Package taskclient bridges the pinned client's summary-only task DTO to the
// modern scheduler detail profile. It shares the configured transport/session.
package taskclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/synology-community/go-synology/pkg/api"
	"github.com/synology-community/go-synology/pkg/api/core"
	"github.com/synology-community/go-synology/pkg/util"
)

const scheduler = "SYNO.Core.TaskScheduler"

// ErrAbsent means an authoritative, complete scheduler inventory omitted the ID.
// No DSM error code (including session-not-found) is treated as task absence.
var ErrAbsent = errors.New("task ID is absent from the complete scheduler inventory")

// Detail includes write-critical server metadata without adding public state.
// Raw nested objects retain extensions during unrelated updates.
type Detail struct {
	ID                           int64
	Name, Owner, RealOwner, Type string
	Enable                       bool
	Schedule                     core.TaskSchedule
	Extra                        core.TaskExtra
	schedule, extra              map[string]json.RawMessage
}

// PreservesMetadata checks fields deliberately carried through an update but
// not owned by the Terraform schema. Values never enter diagnostics.

type API interface {
	Read(context.Context, int64) (*Detail, error)
}

type Client struct {
	client api.Api
	http   *http.Client
}

func New(client api.Api) *Client {
	shared := client.Client().HTTPClient
	return &Client{client: client, http: &http.Client{
		Transport:     shared.Transport,
		Jar:           shared.Jar,
		Timeout:       shared.Timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

// call uses POST with credentials in the body/header, the existing HTTP
// transport and TLS policy, and no retryablehttp wrapper. It neither copies a
// mutex-bearing client nor changes shared retry settings. No body/URL/error text
// from the transport is returned to callers or logged here.
func (c *Client) call(
	ctx context.Context,
	name, method string,
	version int,
	params any,
) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	values, err := util.Query(
		api.Method{API: name, Method: method, Version: version},
		params,
		c.client.Credentials(),
	)
	if err != nil {
		return nil, errors.New("cannot encode scheduler request")
	}
	base := c.client.BaseUrl()
	if base == nil {
		return nil, errors.New("scheduler client has no endpoint")
	}
	endpoint := *base
	endpoint.RawQuery = ""
	endpoint.Fragment = ""
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint.String(),
		strings.NewReader(values.Encode()),
	)
	if err != nil {
		return nil, errors.New("cannot construct scheduler request")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	credentials := c.client.Credentials()
	req.Header.Set("Cookie", credentials.GetCookie())
	if credentials.Token != "" {
		req.Header.Set("X-SYNO-TOKEN", credentials.Token)
	}
	response, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, errors.New(
				"scheduler request cancelled or timed out; inspect the exact task ID before retrying a write",
			)
		}
		return nil, errors.New(
			"scheduler transport failed; inspect TLS/connectivity and the exact task ID before retrying a write",
		)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("scheduler HTTP status %d", response.StatusCode)
	}
	const maxBody = 4 << 20
	body, err := io.ReadAll(io.LimitReader(response.Body, maxBody+1))
	if err != nil || len(body) > maxBody {
		return nil, errors.New("cannot read bounded scheduler response")
	}
	var envelope struct {
		Success *bool           `json:"success"`
		Data    json.RawMessage `json:"data"`
		Error   *struct {
			Code int `json:"code"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &envelope) != nil || envelope.Success == nil {
		return nil, errors.New("malformed scheduler response envelope")
	}
	if *envelope.Success && envelope.Error != nil && envelope.Error.Code != 0 {
		return nil, errors.New("contradictory scheduler response envelope")
	}
	if !*envelope.Success {
		if envelope.Error == nil || envelope.Error.Code == 0 {
			return nil, errors.New("malformed scheduler error response")
		}
		return nil, fmt.Errorf(
			"scheduler %s v%d rejected the request (DSM code %d); check permissions, session and supported API profile",
			method,
			version,
			envelope.Error.Code,
		)
	}
	return envelope.Data, nil
}

func (c *Client) Read(ctx context.Context, id int64) (*Detail, error) {
	if id <= 0 {
		return nil, errors.New("task ID must be positive")
	}
	body, err := c.call(ctx, scheduler, "list", 3, &core.ListTaskRequest{})
	if err != nil {
		return nil, err
	}
	var inventory struct {
		Tasks  []json.RawMessage `json:"tasks"`
		Total  *int64            `json:"total"`
		Offset *int64            `json:"offset"`
	}
	if json.Unmarshal(body, &inventory) != nil || inventory.Tasks == nil ||
		inventory.Total == nil ||
		inventory.Offset == nil || *inventory.Offset != 0 ||
		*inventory.Total != int64(len(inventory.Tasks)) {
		return nil, errors.New("scheduler inventory is incomplete or malformed; state retained")
	}
	var identity *struct {
		ID        *int64  `json:"id"`
		RealOwner *string `json:"real_owner"`
		Type      *string `json:"type"`
	}
	seen := map[int64]bool{}
	for _, row := range inventory.Tasks {
		var entry struct {
			ID        *int64  `json:"id"`
			RealOwner *string `json:"real_owner"`
			Type      *string `json:"type"`
		}
		if json.Unmarshal(row, &entry) != nil || entry.ID == nil || *entry.ID <= 0 ||
			seen[*entry.ID] {
			return nil, errors.New("scheduler inventory has missing or ambiguous task identities")
		}
		seen[*entry.ID] = true
		if *entry.ID == id {
			identity = &entry
		}
	}
	if identity == nil {
		return nil, ErrAbsent
	}
	if identity.RealOwner == nil || *identity.RealOwner == "" || identity.Type == nil ||
		*identity.Type != "script" {
		return nil, errors.New(
			"unsupported task identity/type; only positively identified script tasks can be managed",
		)
	}
	params := struct {
		ID        int64  `url:"id"`
		RealOwner string `url:"real_owner"`
	}{id, *identity.RealOwner}
	body, err = c.call(ctx, scheduler, "get", 4, &params)
	if err != nil {
		return nil, err
	}
	detail, err := Decode(body, id)
	if err != nil {
		return nil, err
	}
	if detail.RealOwner != *identity.RealOwner || detail.Type != *identity.Type {
		return nil, errors.New(
			"task identity changed during detail lookup; refresh before retrying",
		)
	}
	return detail, nil
}

// Decode rejects absent/null managed fields instead of interpreting their Go
// zero values as remote drift. Error text never contains imported script text.
func Decode(body []byte, id int64) (*Detail, error) {
	var wire struct {
		ID        *int64                     `json:"id"`
		Name      *string                    `json:"name"`
		Owner     *string                    `json:"owner"`
		RealOwner *string                    `json:"real_owner"`
		Type      *string                    `json:"type"`
		Enable    *bool                      `json:"enable"`
		Schedule  map[string]json.RawMessage `json:"schedule"`
		Extra     map[string]json.RawMessage `json:"extra"`
	}
	if json.Unmarshal(body, &wire) != nil || wire.ID == nil || *wire.ID != id || wire.Name == nil ||
		*wire.Name == "" ||
		wire.Owner == nil ||
		*wire.Owner == "" ||
		wire.RealOwner == nil ||
		*wire.RealOwner == "" ||
		wire.Type == nil ||
		wire.Enable == nil ||
		wire.Schedule == nil ||
		wire.Extra == nil {
		return nil, errors.New("task detail is incomplete, malformed or has the wrong identity")
	}
	if *wire.Type != "script" {
		return nil, errors.New("unsupported task type; only scheduled script tasks can be managed")
	}
	detail := &Detail{
		ID:        id,
		Name:      *wire.Name,
		Owner:     *wire.Owner,
		RealOwner: *wire.RealOwner,
		Type:      *wire.Type,
		Enable:    *wire.Enable,
		schedule:  wire.Schedule,
		extra:     wire.Extra,
	}
	// Repeat picker choices are UI metadata. Unknown schedule constraints cannot
	// safely be approximated; reject them even when the familiar fields decode.
	for key := range wire.Schedule {
		switch key {
		case "date_type", "date", "week_day", "hour", "minute", "monthly_week",
			"repeat_hour", "repeat_min", "repeat_date", "repeat_min_store_config",
			"repeat_hour_store_config", "last_work_hour":
		default:
			return nil, errors.New(
				"unsupported scheduler calendar field; use a supported daily/weekly or interval schedule",
			)
		}
	}
	for _, key := range []string{"date_type", "hour", "minute", "repeat_date", "repeat_hour", "repeat_min", "week_day"} {
		value, ok := wire.Schedule[key]
		if !ok || string(value) == "null" {
			return nil, errors.New("task schedule is missing a required calendar field")
		}
	}
	for _, key := range []string{"script", "notify_enable", "notify_if_error", "notify_mail"} {
		value, ok := wire.Extra[key]
		if !ok || string(value) == "null" {
			return nil, errors.New(
				"task detail is missing script or notification fields; this API profile is unsupported",
			)
		}
	}
	scheduleJSON, _ := json.Marshal(wire.Schedule)
	extraJSON, _ := json.Marshal(wire.Extra)
	if json.Unmarshal(scheduleJSON, &detail.Schedule) != nil ||
		json.Unmarshal(extraJSON, &detail.Extra) != nil {
		return nil, errors.New("malformed task schedule or script/notification details")
	}
	return detail, nil
}
