package vulcanizer

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

// ServerSetup type contains the Method, Path, Body and Response strings, as well as the HTTP Status code.
type ServerSetup struct {
	Method, Path, Body, Response string
	HTTPStatus                   int
}

// setupTestServer sets up an HTTP test server to serve data to the test that come after it.

func setupTestServers(t *testing.T, setups []*ServerSetup) (string, int, *httptest.Server) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		requestBytes, _ := ioutil.ReadAll(r.Body)
		requestBody := string(requestBytes)

		matched := false
		for _, setup := range setups {
			if r.Method == setup.Method && r.URL.EscapedPath() == setup.Path && requestBody == setup.Body {
				matched = true
				if setup.HTTPStatus == 0 {
					w.WriteHeader(http.StatusOK)
				} else {
					w.WriteHeader(setup.HTTPStatus)
				}
				w.Write([]byte(setup.Response))
			}
		}

		if matched == false {
			t.Fatalf("No requests matched setup. Got method %s, Path %s, body %s", r.Method, r.URL.EscapedPath(), requestBody)
		}
	}))
	url, _ := url.Parse(ts.URL)
	port, _ := strconv.Atoi(url.Port())
	return url.Hostname(), port, ts
}

// TestSetSetting Func is an integration test for all things that use the SetSetting functionality.
func TestSetSetting(t *testing.T) {

	tt := []struct {
		Name        string
		Method      string
		Path        string
		Body        string
		GetResponse string
		PutResponse string
		Setting     string
		SetValue    string
		HTTPStatus  int
		OldValue    string
	}{
		{
			// Tests for behavior with existing transient setting.
			Name:        "Existing Transient Setting",
			Path:        "/_cluster/settings",
			GetResponse: `{"persistent":{},"transient":{"cluster":{"routing":{"allocation":{"exclude":{"_name":"10.0.0.2"}}}}}}`,
			Body:        `{"transient":{"cluster.routing.allocation.exclude._name":"10.0.0.99"}}`,
			PutResponse: `{"persistent":{},"transient":{"cluster":{"routing":{"allocation":{"exclude":{"_name":"10.0.0.99"}}}}}}`,
			Setting:     "cluster.routing.allocation.exclude._name",
			SetValue:    "10.0.0.99",
			OldValue:    "10.0.0.2",
		},

		{
			// Tests for behavior with existing persistent settings.
			Name:        "Existing Persistent Setting",
			Path:        "/_cluster/settings",
			GetResponse: `{"transient":{},"persistent":{"cluster":{"routing":{"allocation":{"exclude":{"_name":"10.0.0.2"}}}}}}`,
			Body:        `{"transient":{"cluster.routing.allocation.exclude._name":"10.0.0.99"}}`,
			PutResponse: `{"persistent":{},"transient":{"cluster":{"routing":{"allocation":{"exclude":{"_name":"10.0.0.99"}}}}}}`,
			Setting:     "cluster.routing.allocation.exclude._name",
			SetValue:    "10.0.0.99",
			OldValue:    "10.0.0.2",
		},

		{
			// Tests for behavior with NO existing persistent settings.
			Name:        "No existing settings",
			Path:        "/_cluster/settings",
			GetResponse: `{"transient":{},"persistent":{}}`,
			Body:        `{"transient":{"cluster.routing.allocation.exclude._name":"10.0.0.99"}}`,
			PutResponse: `{"persistent":{},"transient":{"cluster":{"routing":{"allocation":{"exclude":{"_name":"10.0.0.99"}}}}}}`,
			Setting:     "cluster.routing.allocation.exclude._name",
			SetValue:    "10.0.0.99",
			OldValue:    "sss",
		},
	}

	for _, x := range tt {
		t.Run(fmt.Sprintf("%s", x.Name), func(st *testing.T) {

			getSetup := &ServerSetup{
				Method:   "GET",
				Path:     "/_cluster/settings",
				Response: x.GetResponse,
			}
			putSetup := &ServerSetup{
				Method:   "PUT",
				Path:     "/_cluster/settings",
				Body:     x.Body,
				Response: x.PutResponse,
			}

			host, port, ts := setupTestServers(t, []*ServerSetup{getSetup, putSetup})
			defer ts.Close()

			oldSetting, newSetting, err := SetSetting(host, port, x.Setting, x.SetValue)

			if err != nil {
				st.Errorf("Expected error to be nil, %s", err)
			}

			if oldSetting != x.OldValue {
				st.Errorf("Unexpected old value, got %s", oldSetting)
			}

			if newSetting != "10.0.0.99" {
				st.Errorf("Unexpected new value, got %s", newSetting)
			}

		})
	}
}
