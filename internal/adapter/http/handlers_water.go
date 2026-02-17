package adapthttp

import (
	"net/http"
	"time"
)

func (s *Server) handleWaterToday(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	user := userFromContext(r)
	today := localDayString(time.Now())
	total, err := s.water.GetTodayTotal(r.Context(), user.ID, today)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"today": today, "totalLiters": total})
}

func (s *Server) handleWaterEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	user := userFromContext(r)
	var body struct {
		DeltaLiters float64 `json:"deltaLiters"`
	}
	if err := parseJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	id, err := s.water.RecordEvent(r.Context(), user.ID, body.DeltaLiters)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id})
}

func (s *Server) handleWaterRecent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	user := userFromContext(r)
	limit := intQuery(r, "limit", 20)
	items, err := s.water.ListRecent(r.Context(), user.ID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleWaterUndoLast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	user := userFromContext(r)
	undone, id, err := s.water.UndoLast(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"undone": undone, "id": id})
}
