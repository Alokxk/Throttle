package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Alokxk/Throttle/config"
	"github.com/Alokxk/Throttle/db"
	"github.com/Alokxk/Throttle/handlers"
	"github.com/Alokxk/Throttle/middleware"
)

var h *handlers.Handler
var apiKey string
var clientID string

func TestMain(m *testing.M) {
	cfg := config.Load()

	pgDB := db.NewPostgresDB(cfg.DatabaseURL)
	redisClient := db.NewRedisClient(cfg.RedisURL)

	h = &handlers.Handler{
		DB:    pgDB,
		Redis: redisClient,
	}

	setupTestClient()
	os.Exit(m.Run())
}

func setupTestClient() {
	body := bytes.NewBufferString(`{"name":"test-suite","email":"testsuite@throttle.dev"}`)
	req := httptest.NewRequest(http.MethodPost, "/register", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Register(w, req)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if w.Code == http.StatusCreated {
		apiKey = resp["api_key"].(string)
		clientID = resp["client_id"].(string)
		return
	}

	if w.Code == http.StatusConflict {
		fetchExistingClient()
		return
	}
}

func fetchExistingClient() {
	row := h.DB.QueryRow(`
		SELECT api_key, id FROM clients 
		WHERE email = 'testsuite@throttle.dev' AND is_active = true
	`)
	row.Scan(&apiKey, &clientID)
}

func makeCheckRequest(identifier, algorithm string, limit, window int) *httptest.ResponseRecorder {
	body := fmt.Sprintf(`{
		"identifier": "%s",
		"limit": %d,
		"window": %d,
		"algorithm": "%s"
	}`, identifier, limit, window, algorithm)

	req := httptest.NewRequest(http.MethodPost, "/check", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)
	w := httptest.NewRecorder()

	middleware.Auth(h.DB, h.Check)(w, req)
	return w
}

// Registration tests

func TestRegister_Success(t *testing.T) {
	body := bytes.NewBufferString(`{"name":"new-app","email":"newapp@throttle.dev"}`)
	req := httptest.NewRequest(http.MethodPost, "/register", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Register(w, req)

	if w.Code != http.StatusCreated && w.Code != http.StatusConflict {
		t.Errorf("expected 201 or 409, got %d", w.Code)
	}
}

func TestRegister_MissingFields(t *testing.T) {
	body := bytes.NewBufferString(`{"name":""}`)
	req := httptest.NewRequest(http.MethodPost, "/register", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Register(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRegister_InvalidEmail(t *testing.T) {
	body := bytes.NewBufferString(`{"name":"app","email":"notanemail"}`)
	req := httptest.NewRequest(http.MethodPost, "/register", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Register(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// Auth middleware tests

func TestAuth_MissingKey(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/check", nil)
	w := httptest.NewRecorder()

	middleware.Auth(h.DB, h.Check)(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuth_InvalidKey(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/check", nil)
	req.Header.Set("X-API-Key", "thr_invalid000000000000000")
	w := httptest.NewRecorder()

	middleware.Auth(h.DB, h.Check)(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// Fixed window tests

func TestFixedWindow_AllowsUnderLimit(t *testing.T) {
	w := makeCheckRequest("fw_test_allow", "fixed_window", 10, 60)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["allowed"] != true {
		t.Errorf("expected allowed=true, got %v", resp["allowed"])
	}
}

func TestFixedWindow_RejectsOverLimit(t *testing.T) {
	identifier := "fw_test_reject"
	limit := 3

	for i := 0; i < limit; i++ {
		makeCheckRequest(identifier, "fixed_window", limit, 60)
	}

	w := makeCheckRequest(identifier, "fixed_window", limit, 60)
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["allowed"] != false {
		t.Errorf("expected allowed=false after exceeding limit, got %v", resp["allowed"])
	}
}

func TestFixedWindow_Headers(t *testing.T) {
	w := makeCheckRequest("fw_test_headers", "fixed_window", 10, 60)

	headers := []string{
		"X-Ratelimit-Limit",
		"X-Ratelimit-Remaining",
		"X-Ratelimit-Reset",
		"X-Ratelimit-Algorithm",
	}

	for _, h := range headers {
		if w.Header().Get(h) == "" {
			t.Errorf("missing header: %s", h)
		}
	}
}

// Sliding window tests

func TestSlidingWindow_AllowsUnderLimit(t *testing.T) {
	w := makeCheckRequest("sw_test_allow", "sliding_window", 10, 60)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["allowed"] != true {
		t.Errorf("expected allowed=true, got %v", resp["allowed"])
	}
}

func TestSlidingWindow_RejectsOverLimit(t *testing.T) {
	identifier := "sw_test_reject"
	limit := 3

	for i := 0; i < limit; i++ {
		makeCheckRequest(identifier, "sliding_window", limit, 60)
	}

	w := makeCheckRequest(identifier, "sliding_window", limit, 60)
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["allowed"] != false {
		t.Errorf("expected allowed=false after exceeding limit, got %v", resp["allowed"])
	}
}

// Token bucket tests

func TestTokenBucket_AllowsUnderCapacity(t *testing.T) {
	w := makeCheckRequest("tb_test_allow", "token_bucket", 10, 0)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["allowed"] != true {
		t.Errorf("expected allowed=true, got %v", resp["allowed"])
	}
}

func TestTokenBucket_RejectsWhenEmpty(t *testing.T) {
	identifier := "tb_test_reject"
	capacity := 3

	for i := 0; i < capacity; i++ {
		makeCheckRequest(identifier, "token_bucket", capacity, 0)
	}

	w := makeCheckRequest(identifier, "token_bucket", capacity, 0)
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["allowed"] != false {
		t.Errorf("expected allowed=false when bucket empty, got %v", resp["allowed"])
	}
}

func TestTokenBucket_RetryAfterOnRejection(t *testing.T) {
	identifier := "tb_test_retry"
	capacity := 2

	for i := 0; i < capacity; i++ {
		makeCheckRequest(identifier, "token_bucket", capacity, 0)
	}

	w := makeCheckRequest(identifier, "token_bucket", capacity, 0)

	if w.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header on rejected token bucket request")
	}
}

// Validation tests

func TestCheck_MissingIdentifier(t *testing.T) {
	body := bytes.NewBufferString(`{"limit":10,"window":60,"algorithm":"fixed_window"}`)
	req := httptest.NewRequest(http.MethodPost, "/check", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)
	w := httptest.NewRecorder()

	middleware.Auth(h.DB, h.Check)(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCheck_InvalidAlgorithm(t *testing.T) {
	body := bytes.NewBufferString(`{"identifier":"u1","limit":10,"window":60,"algorithm":"magic"}`)
	req := httptest.NewRequest(http.MethodPost, "/check", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)
	w := httptest.NewRecorder()

	middleware.Auth(h.DB, h.Check)(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
