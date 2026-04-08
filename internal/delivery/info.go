package delivery

import "net/http"

// handleInfo godoc
// @Summary      Health check
// @Description  Всегда возвращает 200 OK.
// @Tags         System
// @Produce      json
// @Success      200 {object} object{status=string}
// @Router       /_info [get]
func handleInfo(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
