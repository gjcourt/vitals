package adapthttp

import (
	"net/http"
	"time"
)

func (s *Server) handleChartsDaily(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	user := userFromContext(r)
	days := intQuery(r, "days", 90)
	unit := r.URL.Query().Get("unit")
	if unit == "" {
		unit = "lb"
	}

	points, err := s.charts.GetDaily(r.Context(), user.ID, days, unit)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"days":  days,
		"unit":  unit,
		"today": localDayString(time.Now()),
		"items": points,
	})
}
