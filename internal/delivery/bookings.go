package delivery

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"room-booking/internal/domain"
)

// handleCreateBooking godoc
// @Summary      Создать бронь на слот
// @Description  Создаёт бронь. Только user. userId берётся из JWT.
// @Tags         Bookings
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body object{slotId=string,createConferenceLink=bool} true "Данные брони"
// @Success      201 {object} object{booking=domain.Booking}
// @Failure      400 {object} object{error=object{code=string,message=string}}
// @Failure      401 {object} object{error=object{code=string,message=string}}
// @Failure      403 {object} object{error=object{code=string,message=string}}
// @Failure      404 {object} object{error=object{code=string,message=string}}
// @Failure      409 {object} object{error=object{code=string,message=string}}
// @Failure      500 {object} object{error=object{code=string,message=string}}
// @Router       /bookings/create [post]
func handleCreateBooking(bookingUC BookingUsecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)

		var req struct {
			SlotID               string `json:"slotId"`
			CreateConferenceLink bool   `json:"createConferenceLink"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
			return
		}

		slotID, err := uuid.Parse(req.SlotID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid slot_id")
			return
		}

		booking, err := bookingUC.CreateBooking(r.Context(), userID, slotID, req.CreateConferenceLink)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrSlotNotFound):
				writeError(w, http.StatusNotFound, "SLOT_NOT_FOUND", err.Error())
			case errors.Is(err, domain.ErrSlotBooked):
				writeError(w, http.StatusConflict, "SLOT_ALREADY_BOOKED", err.Error())
			case errors.Is(err, domain.ErrInvalidRequest):
				writeError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
			default:
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
			}
			return
		}

		writeJSON(w, http.StatusCreated, map[string]interface{}{"booking": booking})
	}
}

// handleListBookings godoc
// @Summary      Список всех броней с пагинацией
// @Description  Возвращает все брони. Только admin.
// @Tags         Bookings
// @Produce      json
// @Security     BearerAuth
// @Param        page     query int false "Номер страницы (по умолчанию 1)"     minimum(1)
// @Param        pageSize query int false "Размер страницы (по умолчанию 20, макс 100)" minimum(1) maximum(100)
// @Success      200 {object} object{bookings=[]domain.Booking,pagination=domain.Pagination}
// @Failure      401 {object} object{error=object{code=string,message=string}}
// @Failure      403 {object} object{error=object{code=string,message=string}}
// @Failure      500 {object} object{error=object{code=string,message=string}}
// @Router       /bookings/list [get]
func handleListBookings(bookingUC BookingUsecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page := 1
		pageSize := 20

		if p := r.URL.Query().Get("page"); p != "" {
			if v, err := strconv.Atoi(p); err == nil && v > 0 {
				page = v
			}
		}
		if ps := r.URL.Query().Get("pageSize"); ps != "" {
			if v, err := strconv.Atoi(ps); err == nil && v > 0 {
				if v > 100 {
					v = 100
				}
				pageSize = v
			}
		}

		bookings, pagination, err := bookingUC.ListAll(r.Context(), page, pageSize)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"bookings":   bookings,
			"pagination": pagination,
		})
	}
}

// handleListMyBookings godoc
// @Summary      Список броней текущего пользователя
// @Description  Возвращает только будущие брони пользователя из JWT. Только user.
// @Tags         Bookings
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} object{bookings=[]domain.Booking}
// @Failure      401 {object} object{error=object{code=string,message=string}}
// @Failure      403 {object} object{error=object{code=string,message=string}}
// @Failure      500 {object} object{error=object{code=string,message=string}}
// @Router       /bookings/my [get]
func handleListMyBookings(bookingUC BookingUsecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)

		bookings, err := bookingUC.ListMy(r.Context(), userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{"bookings": bookings})
	}
}

// handleCancelBooking godoc
// @Summary      Отменить бронь
// @Description  Идемпотентно отменяет бронь. Только user, только свою бронь.
// @Tags         Bookings
// @Produce      json
// @Security     BearerAuth
// @Param        bookingId path string true "ID брони" format(uuid)
// @Success      200 {object} object{booking=domain.Booking}
// @Failure      401 {object} object{error=object{code=string,message=string}}
// @Failure      403 {object} object{error=object{code=string,message=string}}
// @Failure      404 {object} object{error=object{code=string,message=string}}
// @Failure      500 {object} object{error=object{code=string,message=string}}
// @Router       /bookings/{bookingId}/cancel [post]
func handleCancelBooking(bookingUC BookingUsecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)

		bookingIDStr := chi.URLParam(r, "bookingId")
		bookingID, err := uuid.Parse(bookingIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid booking id")
			return
		}

		booking, err := bookingUC.CancelBooking(r.Context(), userID, bookingID)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrBookingNotFound):
				writeError(w, http.StatusNotFound, "BOOKING_NOT_FOUND", err.Error())
			case errors.Is(err, domain.ErrForbidden):
				writeError(w, http.StatusForbidden, "FORBIDDEN", err.Error())
			default:
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
			}
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{"booking": booking})
	}
}
