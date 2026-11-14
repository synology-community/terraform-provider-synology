package taskclient

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/synology-community/go-synology"
	"github.com/synology-community/go-synology/pkg/api"
)

func fixture(t *testing.T) []byte {
	t.Helper()
	body, err := os.ReadFile("testdata/script.json")
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func object(t *testing.T) map[string]any {
	t.Helper()
	var obj map[string]any
	if err := json.Unmarshal(fixture(t), &obj); err != nil {
		t.Fatal(err)
	}
	return obj
}

func envelope(t *testing.T, w http.ResponseWriter, data any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"success": true, "data": data}); err != nil {
		t.Error(err)
	}
}

func testClient(t *testing.T, handler http.HandlerFunc) (*Client, synology.Api) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	shared, err := synology.New(
		api.Options{Host: server.URL, AllowHTTP: true, VerifyCert: true, RetryLimit: 5},
	)
	if err != nil {
		t.Fatal(err)
	}
	shared.Client().RetryMax = 5
	return New(shared), shared
}

func form(t *testing.T, r *http.Request) url.Values {
	t.Helper()
	if r.Method != "POST" || r.URL.RawQuery != "" {
		t.Error("scheduler request is not a query-free POST")
	}
	if err := r.ParseForm(); err != nil {
		t.Error(err)
	}
	return r.Form
}

func TestReadExactIdentityAndFullDetail(t *testing.T) {
	calls := []string{}
	c, shared := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		values := form(t, r)
		calls = append(calls, values.Get("method"))
		switch values.Get("method") {
		case "list":
			if values.Get("version") != "3" {
				t.Error("wrong list version")
			}
			envelope(
				t,
				w,
				map[string]any{
					"tasks": []any{
						map[string]any{
							"id":         41,
							"name":       "same name",
							"real_owner": "other",
							"type":       "script",
						},
						map[string]any{
							"id":         42,
							"name":       "same name",
							"real_owner": "root",
							"type":       "script",
						},
					},
					"total": 2,
				},
			)
		case "get":
			if values.Get("id") != "42" || values.Get("real_owner") != "root" ||
				values.Get("version") != "4" {
				t.Error("get did not use exact identity/profile")
			}
			envelope(t, w, json.RawMessage(fixture(t)))
		default:
			t.Error("read dispatched a mutation")
		}
	})
	detail, err := c.Read(t.Context(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Extra.Script != "/usr/local/sbin/example-maintenance" || detail.Schedule.Hour != 3 ||
		detail.Enable ||
		detail.Owner != "runner" ||
		!detail.Extra.NotifyEnable {
		t.Fatal("detail projection incomplete")
	}
	if shared.Client().RetryMax != 5 {
		t.Fatal("read changed the shared retry policy")
	}
	if strings.Join(calls, ",") != "list,get" {
		t.Fatal(calls)
	}
}

func TestReadOnlyCompleteInventoryProvesAbsence(t *testing.T) {
	tests := []struct {
		name, body string
		absent     bool
	}{
		{"empty complete", `{"success":true,"data":{"tasks":[],"total":0,"offset":0}}`, true},
		{"complete without offset", `{"success":true,"data":{"tasks":[],"total":0}}`, true},
		{"null offset", `{"success":true,"data":{"tasks":[],"total":0,"offset":null}}`, false},
		{"malformed offset", `{"success":true,"data":{"tasks":[],"total":0,"offset":"0"}}`, false},
		{"missing tasks", `{"success":true,"data":{"total":0}}`, false},
		{"null tasks", `{"success":true,"data":{"tasks":null,"total":0}}`, false},
		{"missing total", `{"success":true,"data":{"tasks":[]}}`, false},
		{"partial", `{"success":true,"data":{"tasks":[],"total":1,"offset":0}}`, false},
		{"later page", `{"success":true,"data":{"tasks":[],"total":0,"offset":1}}`, false},
		{
			"ambiguous",
			`{"success":true,"data":{"tasks":[{"id":42},{"id":42}],"total":2,"offset":0}}`,
			false,
		},
		{"missing id", `{"success":true,"data":{"tasks":[{}],"total":1,"offset":0}}`, false},
		{
			"unsupported",
			`{"success":true,"data":{"tasks":[{"id":42,"type":"reboot","real_owner":"root"}],"total":1,"offset":0}}`,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := 0
			c, _ := testClient(
				t,
				func(w http.ResponseWriter, r *http.Request) { count++; _, _ = io.WriteString(w, tt.body) },
			)
			_, err := c.Read(t.Context(), 42)
			if err == nil || errors.Is(err, ErrAbsent) != tt.absent || count != 1 {
				t.Fatalf("absence=%v error=%v count=%d", tt.absent, err, count)
			}
		})
	}
}

func TestDecodePreservesCalendarVersion(t *testing.T) {
	obj := object(t)
	nestedObject(t, obj, "schedule")["version"] = 1
	body, err := json.Marshal(obj)
	if err != nil {
		t.Fatal(err)
	}
	detail, err := Decode(body, 42)
	if err != nil {
		t.Fatal(err)
	}
	if string(detail.schedule["version"]) != "1" {
		t.Fatal("calendar version metadata was discarded")
	}
}

func TestDecodeRejectsIncompleteUnsupportedAndNullDetails(t *testing.T) {
	for _, key := range []string{"id", "name", "owner", "real_owner", "enable", "type", "schedule", "extra"} {
		for _, null := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/null=%v", key, null), func(t *testing.T) {
				obj := object(t)
				if null {
					obj[key] = nil
				} else {
					delete(obj, key)
				}
				body, err := json.Marshal(obj)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := Decode(body, 42); err == nil {
					t.Fatal("incomplete detail accepted")
				}
			})
		}
	}
	for _, parent := range []string{"schedule", "extra"} {
		keys := []string{
			"date_type",
			"hour",
			"minute",
			"repeat_date",
			"repeat_hour",
			"repeat_min",
			"week_day",
		}
		if parent == "extra" {
			keys = []string{"script", "notify_enable", "notify_if_error", "notify_mail"}
		}
		for _, key := range keys {
			t.Run(parent+"/"+key, func(t *testing.T) {
				obj := object(t)
				nestedObject(t, obj, parent)[key] = nil
				body, err := json.Marshal(obj)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := Decode(body, 42); err == nil {
					t.Fatal("null managed field accepted")
				}
			})
		}
	}
	for _, mutate := range []func(map[string]any){
		func(obj map[string]any) { obj["id"] = 43 },
		func(obj map[string]any) { obj["type"] = "service" },
		func(obj map[string]any) { nestedObject(t, obj, "schedule")["month"] = 4 },
		func(obj map[string]any) { nestedObject(t, obj, "extra")["script"] = false },
	} {
		obj := object(t)
		mutate(obj)
		body, err := json.Marshal(obj)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Decode(body, 42); err == nil {
			t.Fatal("unsupported or malformed detail accepted")
		}
	}
}

func TestDenialsMalformedResponsesAndNoReplay(t *testing.T) {
	for _, code := range []int{102, 103, 104, 105, 106, 107, 119, 403, 404, 4801} {
		t.Run(fmt.Sprintf("DSM%d", code), func(t *testing.T) {
			count := 0
			c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				count++
				_, _ = fmt.Fprintf(
					w,
					`{"success":false,"error":{"code":%d,"errors":{"token":"SYNTHETIC_PRIVATE_VALUE"}}}`,
					code,
				)
			})
			_, err := c.Read(t.Context(), 42)
			if err == nil || errors.Is(err, ErrAbsent) ||
				strings.Contains(err.Error(), "SYNTHETIC_PRIVATE_VALUE") ||
				count != 1 {
				t.Fatalf("read failure/fallback contract: error=%v count=%d", err, count)
			}
		})
	}
	for _, body := range []string{`not JSON SYNTHETIC_PRIVATE_VALUE`, `{}`, `{"success":false}`, `{"success":true,"data":null}`, `{"success":true,"data":{"tasks":[]}}`, `{"success":true} trailing SYNTHETIC_PRIVATE_VALUE`} {
		t.Run("malformed", func(t *testing.T) {
			count := 0
			c, _ := testClient(
				t,
				func(w http.ResponseWriter, r *http.Request) { count++; _, _ = io.WriteString(w, body) },
			)
			_, err := c.Read(t.Context(), 42)
			if err == nil || errors.Is(err, ErrAbsent) ||
				strings.Contains(err.Error(), "SYNTHETIC_PRIVATE_VALUE") ||
				count != 1 {
				t.Fatalf("unsafe parse error=%v count=%d", err, count)
			}
		})
	}
	for _, status := range []int{403, 404, 429, 500, 503} {
		t.Run(fmt.Sprintf("HTTP%d", status), func(t *testing.T) {
			count := 0
			c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				count++
				w.WriteHeader(status)
				_, _ = io.WriteString(w, "SYNTHETIC_PRIVATE_VALUE")
			})
			_, err := c.Read(t.Context(), 42)
			if err == nil || count != 1 ||
				strings.Contains(err.Error(), "SYNTHETIC_PRIVATE_VALUE") {
				t.Fatal("HTTP failure replayed/leaked")
			}
		})
	}
}

type failingTransport struct {
	err   error
	count int
}

func (f *failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	f.count++
	return nil, f.err
}

func TestTransportAndCancellationNeverProveAbsence(t *testing.T) {
	for _, failure := range []error{x509.UnknownAuthorityError{}, context.DeadlineExceeded, errors.New("SYNTHETIC_PRIVATE_VALUE")} {
		transport := &failingTransport{err: failure}
		shared, err := synology.New(
			api.Options{Host: "https://scheduler.example.test", VerifyCert: true, RetryLimit: 5},
		)
		if err != nil {
			t.Fatal(err)
		}
		var logged bytes.Buffer
		shared.Client().Logger = log.New(&logged, "", 0)
		c := New(shared)
		c.http.Transport = transport
		_, err = c.Read(t.Context(), 42)
		if err == nil || errors.Is(err, ErrAbsent) ||
			strings.Contains(err.Error(), "SYNTHETIC_PRIVATE_VALUE") ||
			transport.count != 1 ||
			logged.Len() != 0 {
			t.Fatal("unsafe transport failure")
		}
	}
}

func TestRedirectNotFollowed(t *testing.T) {
	count := 0
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		count++
		http.Redirect(w, r, "/another-endpoint", http.StatusTemporaryRedirect)
	})
	_, err := c.Read(t.Context(), 42)
	if err == nil || count != 1 {
		t.Fatal("scheduler write followed a redirect")
	}
}

func nestedObject(t *testing.T, obj map[string]any, key string) map[string]any {
	t.Helper()
	nested, ok := obj[key].(map[string]any)
	if !ok {
		t.Fatalf("fixture field %s is not an object", key)
	}
	return nested
}
