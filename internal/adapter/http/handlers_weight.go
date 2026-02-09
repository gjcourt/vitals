package adapthttp

import (
	"net/http"
	"time"
)

func (s *Server) handleWeightToday(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	today := localDayString(time.Now())

	switch r.Method {
	case http.MethodGet:
		entry, err := s.weight.GetTodayWeight(ctx, today)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"today": today, "entry": entry})

	case http.MethodPut:
		var body struct {
			Value float64 `json:"value"`
			Unit  string  `json:"unit"`
		}
		if err := parseJSON(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		entry, _, err := s.weight.RecordWeight(ctx, body.Value, body.Unit)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"today": today, "entry": entry})

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleWeightRecent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	limit := intQuery(r, "limit", 14)
	items, err := s.weight.ListRecent(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleWeightUndoLast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	deleted, entry, today, err := s.weight.UndoLast(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "deleted": deleted, "today": today, "entry": entry})
}
