package delivery

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"room-booking/internal/domain"
)

// handleCreateSchedule godoc
// @Summary      Создать расписание переговорки
// @Description  Создаёт расписание. Только admin. Можно создать только один раз.
// @Tags         Schedules
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        roomId path string true "ID переговорки" format(uuid)
// @Param        body body object{daysOfWeek=[]int,startTime=string,endTime=string} true "Расписание"
// @Success      201 {object} object{schedule=domain.Schedule}
// @Failure      400 {object} object{error=object{code=string,message=string}}
// @Failure      401 {object} object{error=object{code=string,message=string}}
// @Failure      403 {object} object{error=object{code=string,message=string}}
// @Failure      404 {object} object{error=object{code=string,message=string}}
// @Failure      409 {object} object{error=object{code=string,message=string}}
// @Failure      500 {object} object{error=object{code=string,message=string}}
// @Router       /rooms/{roomId}/schedule/create [post]
func handleCreateSchedule(scheduleUC ScheduleUsecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roomIDStr := chi.URLParam(r, "roomId")
		roomID, err := uuid.Parse(roomIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid room id")
			return
		}

		var req struct {
			DaysOfWeek []int  `json:"daysOfWeek"`
			StartTime  string `json:"startTime"`
			EndTime    string `json:"endTime"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
			return
		}

		if len(req.DaysOfWeek) == 0 || req.StartTime == "" || req.EndTime == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "daysOfWeek, startTime and endTime are required")
			return
		}

		schedule, err := scheduleUC.CreateSchedule(r.Context(), roomID, req.DaysOfWeek, req.StartTime, req.EndTime)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrRoomNotFound):
				writeError(w, http.StatusNotFound, "ROOM_NOT_FOUND", err.Error())
			case errors.Is(err, domain.ErrScheduleExists):
				writeError(w, http.StatusConflict, "SCHEDULE_EXISTS", err.Error())
			case errors.Is(err, domain.ErrInvalidRequest):
				writeError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
			default:
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
			}
			return
		}

		writeJSON(w, http.StatusCreated, map[string]interface{}{"schedule": schedule})
	}
}
