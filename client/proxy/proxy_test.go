package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"fmt"

	"istio.io/pilot/apiserver"
)

type FakeHandler struct {
	wantResponse   string
	wantHeaders    http.Header
	sentHeaders    http.Header
	wantStatusCode int
}

func (f *FakeHandler) HandlerFunc(w http.ResponseWriter, r *http.Request) {
	if reflect.DeepEqual(r.Header, f.sentHeaders) {
		_, _ = w.Write([]byte(fmt.Sprintf("Received unexpected headers, wanted: %v got %v",
			f.sentHeaders, r.Header)))
	}
	w.WriteHeader(f.wantStatusCode)
	for k, v := range f.wantHeaders {
		w.Header().Set(k, v[0])
	}
	_, _ = w.Write([]byte(f.wantResponse))
}

func TestGetAddUpdateDeleteListConfig(t *testing.T) {
	cases := []struct {
		name            string
		function        string
		key             Key
		kind            string
		namespace       string
		config          *apiserver.Config
		wantConfig      *apiserver.Config
		wantConfigSlice []apiserver.Config
		sentHeaders     http.Header
		wantHeaders     http.Header
		wantStatus      int
		wantError       bool
	}{
		{
			name:        "TestConfigGet",
			function:    "get",
			key:         Key{Name: "name", Namespace: "namespace", Kind: "route-rule"},
			wantConfig:  &apiserver.Config{Type: "type", Name: "name", Spec: "spec"},
			wantHeaders: http.Header{"Content-Type": []string{"application/json"}},
			wantStatus:  http.StatusOK,
		},
		{
			name:        "TestConfigGetNotFound",
			function:    "get",
			key:         Key{Name: "name", Namespace: "namespace", Kind: "route-rule"},
			wantError:   true,
			wantHeaders: http.Header{"Content-Type": []string{"text/plain"}},
			wantStatus:  http.StatusNotFound,
		},
		{
			name:        "TestConfigGetInvalidConfigType",
			function:    "get",
			key:         Key{Name: "name", Namespace: "namespace", Kind: "route-rule"},
			wantError:   true,
			wantHeaders: http.Header{"Content-Type": []string{"text/plain"}},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "TestConfigGetInvalidRespBody",
			function:    "get",
			key:         Key{Name: "name", Namespace: "namespace", Kind: "route-rule"},
			wantError:   true,
			wantHeaders: http.Header{"Content-Type": []string{"text/plain"}},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "TestConfigAdd",
			function:    "add",
			key:         Key{Name: "name", Namespace: "namespace", Kind: "route-rule"},
			config:      &apiserver.Config{Type: "type", Name: "name", Spec: "spec"},
			wantHeaders: http.Header{"Content-Type": []string{"application/json"}},
			sentHeaders: http.Header{"Content-Type": []string{"application/json"}},
			wantStatus:  http.StatusCreated,
		},
		{
			name:        "TestAddConfigConflict",
			function:    "add",
			key:         Key{Name: "name", Namespace: "namespace", Kind: "route-rule"},
			config:      &apiserver.Config{Type: "type", Name: "name", Spec: "spec"},
			wantError:   true,
			wantHeaders: http.Header{"Content-Type": []string{"text/plain"}},
			sentHeaders: http.Header{"Content-Type": []string{"application/json"}},
			wantStatus:  http.StatusConflict,
		},
		{
			name:        "TestConfigAddInvalidConfigType",
			function:    "add",
			key:         Key{Name: "name", Namespace: "namespace", Kind: "route-rule"},
			config:      &apiserver.Config{Type: "NOTATYPE", Name: "name", Spec: "spec"},
			wantError:   true,
			wantHeaders: http.Header{"Content-Type": []string{"text/plain"}},
			sentHeaders: http.Header{"Content-Type": []string{"application/json"}},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "TestAddConfigInvalidSpec",
			function:    "add",
			key:         Key{Name: "name", Namespace: "namespace", Kind: "route-rule"},
			config:      &apiserver.Config{Type: "type", Name: "name", Spec: "NOTASPEC"},
			wantError:   true,
			wantHeaders: http.Header{"Content-Type": []string{"text/plain"}},
			sentHeaders: http.Header{"Content-Type": []string{"application/json"}},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "TestConfigUpdate",
			function:    "update",
			key:         Key{Name: "name", Namespace: "namespace", Kind: "route-rule"},
			config:      &apiserver.Config{Type: "type", Name: "name", Spec: "spec"},
			wantHeaders: http.Header{"Content-Type": []string{"application/json"}},
			sentHeaders: http.Header{"Content-Type": []string{"application/json"}},
			wantStatus:  http.StatusOK,
		},
		{
			name:        "TestConfigUpdateNotFound",
			function:    "update",
			key:         Key{Name: "name", Namespace: "namespace", Kind: "route-rule"},
			config:      &apiserver.Config{Type: "type", Name: "name", Spec: "spec"},
			wantError:   true,
			wantHeaders: http.Header{"Content-Type": []string{"text/plain"}},
			sentHeaders: http.Header{"Content-Type": []string{"application/json"}},
			wantStatus:  http.StatusNotFound,
		},
		{
			name:        "TestConfigUpdateInvalidConfigType",
			function:    "update",
			key:         Key{Name: "name", Namespace: "namespace", Kind: "route-rule"},
			config:      &apiserver.Config{Type: "NOTATYPE", Name: "name", Spec: "spec"},
			wantError:   true,
			wantHeaders: http.Header{"Content-Type": []string{"text/plain"}},
			sentHeaders: http.Header{"Content-Type": []string{"application/json"}},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "TestUpdateConfigInvalidSpec",
			function:    "update",
			key:         Key{Name: "name", Namespace: "namespace", Kind: "route-rule"},
			config:      &apiserver.Config{Type: "type", Name: "name", Spec: "NOTASPEC"},
			wantError:   true,
			wantHeaders: http.Header{"Content-Type": []string{"text/plain"}},
			sentHeaders: http.Header{"Content-Type": []string{"application/json"}},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "TestConfigDelete",
			function:    "delete",
			key:         Key{Name: "name", Namespace: "namespace", Kind: "route-rule"},
			wantHeaders: http.Header{"Content-Type": []string{"application/json"}},
			wantStatus:  http.StatusOK,
		},
		{
			name:        "TestConfigDeleteNotFound",
			function:    "delete",
			key:         Key{Name: "name", Namespace: "namespace", Kind: "route-rule"},
			wantError:   true,
			wantHeaders: http.Header{"Content-Type": []string{"text/plain"}},
			wantStatus:  http.StatusNotFound,
		},
		{
			name:        "TestConfigDeleteInvalidConfigType",
			function:    "delete",
			key:         Key{Name: "name", Namespace: "namespace", Kind: "route-rule"},
			wantError:   true,
			wantHeaders: http.Header{"Content-Type": []string{"text/plain"}},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:      "TestConfigListWithNamespace",
			function:  "list",
			kind:      "kind",
			namespace: "namespace",
			wantConfigSlice: []apiserver.Config{
				{Type: "type", Name: "name", Spec: "spec"},
				{Type: "type", Name: "name2", Spec: "spec"},
			},
			wantHeaders: http.Header{"Content-Type": []string{"application/json"}},
			wantStatus:  http.StatusOK,
		},
		{
			name:     "TestConfigListWithoutNamespace",
			function: "list",
			kind:     "kind",
			wantConfigSlice: []apiserver.Config{
				{Type: "type", Name: "name", Spec: "spec"},
				{Type: "type", Name: "name2", Spec: "spec"},
			},
			wantHeaders: http.Header{"Content-Type": []string{"application/json"}},
			wantStatus:  http.StatusOK,
		},
		{
			name:        "TestConfigListWithNamespaceInvalidConfigType",
			function:    "list",
			kind:        "kind",
			namespace:   "namespace",
			wantError:   true,
			wantHeaders: http.Header{"Content-Type": []string{"application/json"}},
			wantStatus:  http.StatusBadRequest,
		},
	}
	for _, c := range cases {

		// Setup test server
		var want string
		if c.wantConfigSlice != nil {
			e, _ := json.Marshal(c.wantConfigSlice)
			want = string(e)
		} else {
			e, _ := json.Marshal(c.wantConfig)
			want = string(e)
		}

		fh := &FakeHandler{
			wantResponse:   want,
			wantHeaders:    c.wantHeaders,
			wantStatusCode: c.wantStatus,
			sentHeaders:    c.sentHeaders,
		}
		ts := httptest.NewServer(http.HandlerFunc(fh.HandlerFunc))
		defer ts.Close()

		// Setup Client
		var config *apiserver.Config
		var configSlice []apiserver.Config
		var err error
		client := NewConfigClient(&BasicHTTPRequester{
			BaseURL: ts.URL,
			Client:  &http.Client{Timeout: 1 * time.Second},
		})
		switch c.function {
		case "get":
			config, err = client.GetConfig(c.key)
		case "add":
			err = client.AddConfig(c.key, *c.config)
		case "update":
			err = client.UpdateConfig(c.key, *c.config)
		case "delete":
			err = client.DeleteConfig(c.key)
		case "list":
			configSlice, err = client.ListConfig(c.kind, c.namespace)
		default:
			t.Fatal("didn't supply function to test case, don't know which client function to call")
		}

		// Verify
		if !c.wantError && err != nil {
			t.Errorf("%s: unexpected error: %v", c.name, err)
		} else if c.function == "get" {
			if !reflect.DeepEqual(config, c.wantConfig) {
				t.Errorf("%s: wanted config: %+v, but received: %+v", c.name, c.wantConfig, config)
			}
		} else if c.function == "list" {
			if !reflect.DeepEqual(configSlice, c.wantConfigSlice) {
				t.Errorf("%s: wanted config slice: %+v, but received: %+v", c.name, c.wantConfigSlice, configSlice)
			}
		}
	}
}
