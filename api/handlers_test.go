package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"ds-cache/api/response"
	"ds-cache/internal/cache"
	"ds-cache/internal/cache/lru"
	"ds-cache/internal/cache/tiered"
	"ds-cache/internal/service"

	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	m.Run()
}

// newTestRouter creates the full stack the way main does, so these tests cover
// routing, binding and the service together.
func newTestRouter(t *testing.T) *gin.Engine {
	t.Helper()

	nodes := make([]cache.Store, 0, 3)
	for i := 1; i <= 3; i++ {
		name := "node-" + strconv.Itoa(i)
		nodes = append(nodes, tiered.New(name,
			lru.New(name+"-l1", 2),
			lru.New(name+"-l2", 16),
		))
	}

	cacheService, err := service.NewCacheService(nodes)
	if err != nil {
		t.Fatalf("NewCacheService() error = %v", err)
	}
	return NewRouter(NewHandler(cacheService))
}

// do sends the given request to the router and returns the recorder.
func do(t *testing.T, router *gin.Engine, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// decode returns the response envelope of the given recorder.
func decode(t *testing.T, rec *httptest.ResponseRecorder) response.WebResponse {
	t.Helper()

	var envelope response.WebResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decoding response: %v (body: %s)", err, rec.Body.String())
	}
	return envelope
}

// TestCreateThenGet tests the CreateCache and GetByCacheKey handlers.
func TestCreateThenGet(t *testing.T) {
	router := newTestRouter(t)

	rec := do(t, router, http.MethodPost, "/api/cache-manager/v1", `{"key":"test","value":"value-test"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST status = %d; want %d (body: %s)", rec.Code, http.StatusCreated, rec.Body)
	}

	rec = do(t, router, http.MethodGet, "/api/cache-manager/v1?cacheKey=test", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status = %d; want %d (body: %s)", rec.Code, http.StatusOK, rec.Body)
	}

	data, ok := decode(t, rec).Data.(map[string]any)
	if !ok {
		t.Fatalf("GET data is not an object: %s", rec.Body)
	}
	if data["value"] != "value-test" {
		t.Errorf("GET value = %v; want \"value-test\"", data["value"])
	}
}

// TestCreateExistingKeyReturnsOK tests the CreateCache handler with an
// existing key.
func TestCreateExistingKeyReturnsOK(t *testing.T) {
	router := newTestRouter(t)
	do(t, router, http.MethodPost, "/api/cache-manager/v1", `{"key":"test","value":"v1"}`)

	rec := do(t, router, http.MethodPost, "/api/cache-manager/v1", `{"key":"test","value":"v2"}`)
	if rec.Code != http.StatusOK {
		t.Errorf("POST(existing) status = %d; want %d", rec.Code, http.StatusOK)
	}
}

// TestBothPathFormsAreServed tests that the path is served with and without a
// trailing slash, since a redirected POST would drop its body.
func TestBothPathFormsAreServed(t *testing.T) {
	for _, path := range []string{"/api/cache-manager/v1", "/api/cache-manager/v1/"} {
		t.Run(path, func(t *testing.T) {
			router := newTestRouter(t)

			rec := do(t, router, http.MethodPost, path, `{"key":"test","value":"value-test"}`)
			if rec.Code != http.StatusCreated {
				t.Errorf("POST %s status = %d; want %d", path, rec.Code, http.StatusCreated)
			}
		})
	}
}

// TestGetMissingKeyReturnsNotFound tests the GetByCacheKey handler with a
// non-existing key.
func TestGetMissingKeyReturnsNotFound(t *testing.T) {
	router := newTestRouter(t)

	rec := do(t, router, http.MethodGet, "/api/cache-manager/v1?cacheKey=absent", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("GET(absent) status = %d; want %d", rec.Code, http.StatusNotFound)
	}
}

// TestGetWithoutKeyReturnsBadRequest tests the GetByCacheKey handler with no
// cacheKey parameter.
func TestGetWithoutKeyReturnsBadRequest(t *testing.T) {
	router := newTestRouter(t)

	rec := do(t, router, http.MethodGet, "/api/cache-manager/v1", "")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("GET(no key) status = %d; want %d", rec.Code, http.StatusBadRequest)
	}
}

// TestInvalidBodiesReturnBadRequest tests that a malformed or invalid body is
// answered with 400 and not a panic.
func TestInvalidBodiesReturnBadRequest(t *testing.T) {
	tests := map[string]string{
		"malformed json": `{"key":`,
		"missing value":  `{"key":"test"}`,
		"missing key":    `{"value":"value-test"}`,
		"empty key":      `{"key":"","value":"value-test"}`,
		"key too long":   `{"key":"` + strings.Repeat("k", 201) + `","value":"v"}`,
		"value too long": `{"key":"test","value":"` + strings.Repeat("v", 1001) + `"}`,
	}

	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			router := newTestRouter(t)

			rec := do(t, router, http.MethodPost, "/api/cache-manager/v1", body)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("POST(%s) status = %d; want %d (body: %s)", name, rec.Code, http.StatusBadRequest, rec.Body)
			}
		})
	}
}

// TestDelete tests the DeleteByCacheKey handler.
func TestDelete(t *testing.T) {
	router := newTestRouter(t)
	do(t, router, http.MethodPost, "/api/cache-manager/v1", `{"key":"test","value":"value-test"}`)

	rec := do(t, router, http.MethodDelete, "/api/cache-manager/v1?cacheKey=test", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("DELETE status = %d; want %d", rec.Code, http.StatusOK)
	}

	rec = do(t, router, http.MethodGet, "/api/cache-manager/v1?cacheKey=test", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("GET after delete status = %d; want %d", rec.Code, http.StatusNotFound)
	}

	rec = do(t, router, http.MethodDelete, "/api/cache-manager/v1?cacheKey=test", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("DELETE(absent) status = %d; want %d", rec.Code, http.StatusNotFound)
	}
}

// TestStatsEndpoint tests the Stats handler.
func TestStatsEndpoint(t *testing.T) {
	router := newTestRouter(t)
	do(t, router, http.MethodPost, "/api/cache-manager/v1", `{"key":"test","value":"value-test"}`)
	do(t, router, http.MethodGet, "/api/cache-manager/v1?cacheKey=test", "")

	rec := do(t, router, http.MethodGet, "/api/cache-manager/v1/stats", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /stats status = %d; want %d", rec.Code, http.StatusOK)
	}

	nodes, ok := decode(t, rec).Data.([]any)
	if !ok {
		t.Fatalf("stats data is not a list: %s", rec.Body)
	}
	if len(nodes) != 3 {
		t.Errorf("stats reported %d nodes; want 3", len(nodes))
	}
}

// TestUnknownRoute tests the NoRoute handler.
func TestUnknownRoute(t *testing.T) {
	router := newTestRouter(t)

	rec := do(t, router, http.MethodGet, "/nope", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /nope status = %d; want %d", rec.Code, http.StatusNotFound)
	}
}

// TestHealth tests the health endpoint.
func TestHealth(t *testing.T) {
	router := newTestRouter(t)

	rec := do(t, router, http.MethodGet, "/health", "")
	if rec.Code != http.StatusOK {
		t.Errorf("GET /health status = %d; want %d", rec.Code, http.StatusOK)
	}
}
