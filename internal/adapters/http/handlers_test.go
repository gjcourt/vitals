package adapthttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	adapthttp "vitals/internal/adapters/http"
	"vitals/internal/domain"
	"vitals/internal/ports/inbound"
)

// ---------------------------------------------------------------------------
// Mock repositories (function-fields pattern)
// ---------------------------------------------------------------------------

type mockWeightRepo struct {
	addFn    func(ctx context.Context, userID int64, value float64, unit string, createdAt time.Time) (int64, error)
	deleteFn func(ctx context.Context, userID int64) (bool, error)
	latestFn func(ctx context.Context, userID int64, localDay string) (*domain.WeightEntry, error)
	listFn   func(ctx context.Context, userID int64, limit int) ([]domain.WeightEntry, error)
}

func (m *mockWeightRepo) AddWeightEvent(ctx context.Context, userID int64, value float64, unit string, createdAt time.Time) (int64, error) {
	if m.addFn != nil {
		return m.addFn(ctx, userID, value, unit, createdAt)
	}
	return 1, nil
}

func (m *mockWeightRepo) DeleteLatestWeightEvent(ctx context.Context, userID int64) (bool, error) {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, userID)
	}
	return true, nil
}

func (m *mockWeightRepo) LatestWeightForLocalDay(ctx context.Context, userID int64, localDay string) (*domain.WeightEntry, error) {
	if m.latestFn != nil {
		return m.latestFn(ctx, userID, localDay)
	}
	return &domain.WeightEntry{
		ID: 1, Day: localDay, Value: 80.0, Unit: "kg",
		CreatedAt: time.Now(),
	}, nil
}

func (m *mockWeightRepo) ListRecentWeightEvents(ctx context.Context, userID int64, limit int) ([]domain.WeightEntry, error) {
	if m.listFn != nil {
		return m.listFn(ctx, userID, limit)
	}
	return []domain.WeightEntry{
		{ID: 1, Day: "2026-02-08", Value: 80.0, Unit: "kg", CreatedAt: time.Now()},
	}, nil
}

type mockWaterRepo struct {
	addFn   func(ctx context.Context, userID int64, deltaLiters float64, createdAt time.Time) (int64, error)
	delFn   func(ctx context.Context, userID int64, id int64) error
	listFn  func(ctx context.Context, userID int64, limit int) ([]domain.WaterEvent, error)
	totalFn func(ctx context.Context, userID int64, localDay string) (float64, error)
}

func (m *mockWaterRepo) AddWaterEvent(ctx context.Context, userID int64, deltaLiters float64, createdAt time.Time) (int64, error) {
	if m.addFn != nil {
		return m.addFn(ctx, userID, deltaLiters, createdAt)
	}
	return 42, nil
}

func (m *mockWaterRepo) DeleteWaterEvent(ctx context.Context, userID int64, id int64) error {
	if m.delFn != nil {
		return m.delFn(ctx, userID, id)
	}
	return nil
}

func (m *mockWaterRepo) ListRecentWaterEvents(ctx context.Context, userID int64, limit int) ([]domain.WaterEvent, error) {
	if m.listFn != nil {
		return m.listFn(ctx, userID, limit)
	}
	return []domain.WaterEvent{
		{ID: 10, DeltaLiters: 0.5, CreatedAt: time.Now()},
	}, nil
}

func (m *mockWaterRepo) WaterTotalForLocalDay(ctx context.Context, userID int64, localDay string) (float64, error) {
	if m.totalFn != nil {
		return m.totalFn(ctx, userID, localDay)
	}
	return 2.5, nil
}

// ---------------------------------------------------------------------------
// Test-server helper
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Mock services (inbound port implementations — no app layer import needed)
// ---------------------------------------------------------------------------

type mockWeightService struct{ repo *mockWeightRepo }

func (s *mockWeightService) GetTodayWeight(ctx context.Context, userID int64, today string) (*domain.WeightEntry, error) {
	return s.repo.LatestWeightForLocalDay(ctx, userID, today)
}

func (s *mockWeightService) RecordWeight(ctx context.Context, userID int64, value float64, unit string) (*domain.WeightEntry, string, error) {
	if value <= 0 {
		return nil, "", errors.New("value must be > 0")
	}
	if unit != "kg" && unit != "lb" {
		return nil, "", errors.New("unit must be \"kg\" or \"lb\"")
	}
	_, err := s.repo.AddWeightEvent(ctx, userID, value, unit, time.Now())
	if err != nil {
		return nil, "", err
	}
	return &domain.WeightEntry{Value: value, Unit: unit}, "", nil
}

func (s *mockWeightService) ListRecent(ctx context.Context, userID int64, limit int) ([]domain.WeightEntry, error) {
	return s.repo.ListRecentWeightEvents(ctx, userID, limit)
}

func (s *mockWeightService) UndoLast(ctx context.Context, userID int64) (bool, *domain.WeightEntry, string, error) {
	ok, err := s.repo.DeleteLatestWeightEvent(ctx, userID)
	return ok, nil, "", err
}

type mockWaterService struct{ repo *mockWaterRepo }

func (s *mockWaterService) GetTodayTotal(ctx context.Context, userID int64, today string) (float64, error) {
	return s.repo.WaterTotalForLocalDay(ctx, userID, today)
}

func (s *mockWaterService) RecordEvent(ctx context.Context, userID int64, deltaLiters float64) (int64, error) {
	if deltaLiters == 0 || deltaLiters < -10 || deltaLiters > 10 {
		return 0, errors.New("deltaLiters must be non-zero and within [-10, 10]")
	}
	return s.repo.AddWaterEvent(ctx, userID, deltaLiters, time.Now())
}

func (s *mockWaterService) ListRecent(ctx context.Context, userID int64, limit int) ([]domain.WaterEvent, error) {
	return s.repo.ListRecentWaterEvents(ctx, userID, limit)
}

func (s *mockWaterService) UndoLast(ctx context.Context, userID int64) (bool, int64, error) {
	events, err := s.repo.ListRecentWaterEvents(ctx, userID, 1)
	if err != nil || len(events) == 0 {
		return false, 0, err
	}
	id := events[0].ID
	return true, id, s.repo.DeleteWaterEvent(ctx, userID, id)
}

type mockChartsService struct {
	weight *mockWeightRepo
	water  *mockWaterRepo
}

func (s *mockChartsService) GetDaily(_ context.Context, _ int64, _ int, _ string) ([]inbound.DayPoint, error) {
	return nil, nil
}

type mockAuthService struct{}

func (s *mockAuthService) Login(_ context.Context, _, _, _, _ string) (string, error) {
	return "token", nil
}
func (s *mockAuthService) Logout(_ context.Context, _ string) error { return nil }
func (s *mockAuthService) ValidateSession(_ context.Context, _, _ string) (*domain.User, error) {
	return &domain.User{ID: 1, Username: "test"}, nil
}
func (s *mockAuthService) CreateInitialUser(_ context.Context, _, _ string) error { return nil }
func (s *mockAuthService) ValidateForwardAuth(_ context.Context, _ string) (*domain.User, error) {
	return &domain.User{ID: 1, Username: "test"}, nil
}
func (s *mockAuthService) LoginWithUser(_ context.Context, _, _, _ string) (string, error) {
	return "token", nil
}

func newTestServer(t *testing.T, wr *mockWeightRepo, wa *mockWaterRepo) *httptest.Server {
	t.Helper()

	if wr == nil {
		wr = &mockWeightRepo{}
	}
	if wa == nil {
		wa = &mockWaterRepo{}
	}

	ws := &mockWeightService{repo: wr}
	was := &mockWaterService{repo: wa}
	cs := &mockChartsService{weight: wr, water: wa}
	authSvc := &mockAuthService{}

	webDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(webDir, "index.html"), []byte("<html></html>"), 0o600); err != nil {
		t.Fatal(err)
	}

	srv := adapthttp.New(ws, was, cs, authSvc, webDir).WithoutAuth()
	return httptest.NewServer(srv.Handler())
}

func decodeBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	return m
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestHealthEndpoint(t *testing.T) {
	ts := newTestServer(t, nil, nil)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/health")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body := decodeBody(t, resp)
	if body["ok"] != true {
		t.Fatalf("expected ok=true, got %v", body["ok"])
	}
}

func TestWeightTodayGet(t *testing.T) {
	ts := newTestServer(t, &mockWeightRepo{
		latestFn: func(_ context.Context, _ int64, localDay string) (*domain.WeightEntry, error) {
			return &domain.WeightEntry{
				ID: 1, Day: localDay, Value: 82.3, Unit: "kg",
				CreatedAt: time.Date(2026, 2, 8, 7, 0, 0, 0, time.UTC),
			}, nil
		},
	}, nil)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/weight/today")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body := decodeBody(t, resp)
	if _, ok := body["today"]; !ok {
		t.Fatal("response missing 'today' field")
	}
	if _, ok := body["entry"]; !ok {
		t.Fatal("response missing 'entry' field")
	}
}

func TestWeightTodayPut(t *testing.T) {
	tests := []struct {
		name       string
		payload    map[string]any
		wantStatus int
	}{
		{
			name:       "valid kg",
			payload:    map[string]any{"value": 85.5, "unit": "kg"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "valid lb",
			payload:    map[string]any{"value": 190.0, "unit": "lb"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "value zero",
			payload:    map[string]any{"value": 0, "unit": "kg"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "value negative",
			payload:    map[string]any{"value": -5.0, "unit": "kg"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid unit",
			payload:    map[string]any{"value": 80.0, "unit": "stone"},
			wantStatus: http.StatusBadRequest,
		},
	}

	ts := newTestServer(t, nil, nil)
	defer ts.Close()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b, _ := json.Marshal(tc.payload)
			req, err := http.NewRequest(http.MethodPut, ts.URL+"/api/weight/today", bytes.NewReader(b))
			if err != nil {
				t.Fatalf("new request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close() //nolint:errcheck

			if resp.StatusCode != tc.wantStatus {
				body := decodeBody(t, resp)
				t.Fatalf("expected %d, got %d; body: %v", tc.wantStatus, resp.StatusCode, body)
			}

			if tc.wantStatus == http.StatusOK {
				body := decodeBody(t, resp)
				if _, ok := body["entry"]; !ok {
					t.Fatal("response missing 'entry' field")
				}
			}
		})
	}
}

func TestWeightRecent(t *testing.T) {
	items := []domain.WeightEntry{
		{ID: 1, Day: "2026-02-08", Value: 80.0, Unit: "kg", CreatedAt: time.Now()},
		{ID: 2, Day: "2026-02-07", Value: 81.0, Unit: "kg", CreatedAt: time.Now()},
	}
	ts := newTestServer(t, &mockWeightRepo{
		listFn: func(_ context.Context, _ int64, limit int) ([]domain.WeightEntry, error) {
			if limit < len(items) {
				return items[:limit], nil
			}
			return items, nil
		},
	}, nil)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/weight/recent?limit=5")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body := decodeBody(t, resp)
	arr, ok := body["items"].([]any)
	if !ok {
		t.Fatal("response missing 'items' array")
	}
	if len(arr) != 2 {
		t.Fatalf("expected 2 items, got %d", len(arr))
	}
}

func TestWeightUndoLast(t *testing.T) {
	ts := newTestServer(t, &mockWeightRepo{
		deleteFn: func(_ context.Context, _ int64) (bool, error) {
			return true, nil
		},
	}, nil)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/api/weight/undo-last", "application/json", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body := decodeBody(t, resp)
	if body["ok"] != true {
		t.Fatalf("expected ok=true, got %v", body["ok"])
	}
	if body["deleted"] != true {
		t.Fatalf("expected deleted=true, got %v", body["deleted"])
	}
}

func TestWaterTodayGet(t *testing.T) {
	ts := newTestServer(t, nil, &mockWaterRepo{
		totalFn: func(_ context.Context, _ int64, _ string) (float64, error) {
			return 3.0, nil
		},
	})
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/water/today")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body := decodeBody(t, resp)
	if _, ok := body["today"]; !ok {
		t.Fatal("response missing 'today' field")
	}
	total, ok := body["totalLiters"].(float64)
	if !ok {
		t.Fatal("response missing 'totalLiters' field")
	}
	if total != 3.0 {
		t.Fatalf("expected totalLiters=3.0, got %v", total)
	}
}

func TestWaterEvent(t *testing.T) {
	tests := []struct {
		name       string
		payload    map[string]any
		wantStatus int
	}{
		{
			name:       "valid positive",
			payload:    map[string]any{"deltaLiters": 0.5},
			wantStatus: http.StatusOK,
		},
		{
			name:       "valid negative",
			payload:    map[string]any{"deltaLiters": -0.25},
			wantStatus: http.StatusOK,
		},
		{
			name:       "zero deltaLiters",
			payload:    map[string]any{"deltaLiters": 0},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "too large",
			payload:    map[string]any{"deltaLiters": 11.0},
			wantStatus: http.StatusBadRequest,
		},
	}

	ts := newTestServer(t, nil, nil)
	defer ts.Close()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b, _ := json.Marshal(tc.payload)
			resp, err := http.Post(ts.URL+"/api/water/event", "application/json", bytes.NewReader(b))
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close() //nolint:errcheck

			if resp.StatusCode != tc.wantStatus {
				body := decodeBody(t, resp)
				t.Fatalf("expected %d, got %d; body: %v", tc.wantStatus, resp.StatusCode, body)
			}

			if tc.wantStatus == http.StatusOK {
				body := decodeBody(t, resp)
				if _, ok := body["id"]; !ok {
					t.Fatal("response missing 'id' field")
				}
			}
		})
	}
}

func TestWaterRecent(t *testing.T) {
	events := []domain.WaterEvent{
		{ID: 10, DeltaLiters: 0.5, CreatedAt: time.Now()},
		{ID: 11, DeltaLiters: 0.3, CreatedAt: time.Now()},
	}
	ts := newTestServer(t, nil, &mockWaterRepo{
		listFn: func(_ context.Context, _ int64, limit int) ([]domain.WaterEvent, error) {
			if limit < len(events) {
				return events[:limit], nil
			}
			return events, nil
		},
	})
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/water/recent?limit=10")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body := decodeBody(t, resp)
	arr, ok := body["items"].([]any)
	if !ok {
		t.Fatal("response missing 'items' array")
	}
	if len(arr) != 2 {
		t.Fatalf("expected 2 items, got %d", len(arr))
	}
}

func TestWaterUndoLast(t *testing.T) {
	ts := newTestServer(t, nil, &mockWaterRepo{
		listFn: func(_ context.Context, _ int64, limit int) ([]domain.WaterEvent, error) {
			return []domain.WaterEvent{
				{ID: 99, DeltaLiters: 0.5, CreatedAt: time.Now()},
			}, nil
		},
	})
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/api/water/undo-last", "application/json", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body := decodeBody(t, resp)
	if body["undone"] != true {
		t.Fatalf("expected undone=true, got %v", body["undone"])
	}
	if id, ok := body["id"].(float64); !ok || id != 99 {
		t.Fatalf("expected id=99, got %v", body["id"])
	}
}

func TestMethodNotAllowed(t *testing.T) {
	ts := newTestServer(t, nil, nil)
	defer ts.Close()

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"DELETE weight/today", http.MethodDelete, "/api/weight/today"},
		{"POST weight/recent", http.MethodPost, "/api/weight/recent"},
		{"GET weight/undo-last", http.MethodGet, "/api/weight/undo-last"},
		{"PUT water/today", http.MethodPut, "/api/water/today"},
		{"GET water/event", http.MethodGet, "/api/water/event"},
		{"POST water/recent", http.MethodPost, "/api/water/recent"},
		{"GET water/undo-last", http.MethodGet, "/api/water/undo-last"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, ts.URL+tc.path, nil)
			if err != nil {
				t.Fatalf("new request: %v", err)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close() //nolint:errcheck

			if resp.StatusCode != http.StatusMethodNotAllowed {
				t.Fatalf("expected 405, got %d", resp.StatusCode)
			}
		})
	}
}
