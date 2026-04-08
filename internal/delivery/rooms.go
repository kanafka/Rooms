package delivery

import (
	"encoding/json"
	"net/http"

	"room-booking/internal/domain"
)

// handleListRooms godoc
// @Summary      Список переговорок
// @Description  Возвращает все переговорки. Доступно admin и user.
// @Tags         Rooms
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} object{rooms=[]domain.Room}
// @Failure      401 {object} object{error=object{code=string,message=string}}
// @Failure      500 {object} object{error=object{code=string,message=string}}
// @Router       /rooms/list [get]
func handleListRooms(roomUC RoomUsecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rooms, err := roomUC.ListRooms(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
			return
		}

		if rooms == nil {
			rooms = []*domain.Room{}
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{"rooms": rooms})
	}
}

// handleCreateRoom godoc
// @Summary      Создать переговорку
// @Description  Создаёт переговорку. Только admin.
// @Tags         Rooms
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body object{name=string,description=string,capacity=int} true "Данные переговорки"
// @Success      201 {object} object{room=domain.Room}
// @Failure      400 {object} object{error=object{code=string,message=string}}
// @Failure      401 {object} object{error=object{code=string,message=string}}
// @Failure      403 {object} object{error=object{code=string,message=string}}
// @Failure      500 {object} object{error=object{code=string,message=string}}
// @Router       /rooms/create [post]
func handleCreateRoom(roomUC RoomUsecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name        string  `json:"name"`
			Description *string `json:"description"`
			Capacity    *int    `json:"capacity"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
			return
		}

		if req.Name == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "name is required")
			return
		}

		room, err := roomUC.CreateRoom(r.Context(), req.Name, req.Description, req.Capacity)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
			return
		}

		writeJSON(w, http.StatusCreated, map[string]interface{}{"room": room})
	}
}
