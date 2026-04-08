package delivery

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"room-booking/internal/domain"
)

// handleListSlots godoc
// @Summary      Список доступных слотов по переговорке и дате
// @Description  Возвращает незанятые слоты на указанную дату. Наиболее нагруженный эндпоинт.
// @Tags         Slots
// @Produce      json
// @Security     BearerAuth
// @Param        roomId path  string true "ID переговорки" format(uuid)
// @Param        date   query string true "Дата в формате YYYY-MM-DD"  format(date)
// @Success      200 {object} object{slots=[]domain.Slot}
// @Failure      400 {object} object{error=object{code=string,message=string}}
// @Failure      401 {object} object{error=object{code=string,message=string}}
// @Failure      404 {object} object{error=object{code=string,message=string}}
// @Failure      500 {object} object{error=object{code=string,message=string}}
// @Router       /rooms/{roomId}/slots/list [get]
func handleListSlots(slotUC SlotUsecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roomIDStr := chi.URLParam(r, "roomId")
		roomID, err := uuid.Parse(roomIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid room id")
			return
		}

		dateStr := r.URL.Query().Get("date")
		if dateStr == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "date query parameter is required")
			return
		}

		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "date must be in YYYY-MM-DD format")
			return
		}

		slots, err := slotUC.GenerateAndGetAvailable(r.Context(), roomID, date.UTC())
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrRoomNotFound):
				writeError(w, http.StatusNotFound, "ROOM_NOT_FOUND", err.Error())
			default:
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
			}
			return
		}

		if slots == nil {
			slots = []*domain.Slot{}
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{"slots": slots})
	}
}
