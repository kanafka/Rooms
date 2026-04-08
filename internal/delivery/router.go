package delivery

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"room-booking/internal/domain"
)

type AuthUsecase interface {
	DummyLogin(role string) (string, error)
	Register(ctx context.Context, email, password, role string) (*domain.User, error)
	Login(ctx context.Context, email, password string) (string, error)
	ValidateToken(tokenStr string) (*domain.Claims, error)
}

type RoomUsecase interface {
	CreateRoom(ctx context.Context, name string, description *string, capacity *int) (*domain.Room, error)
	ListRooms(ctx context.Context) ([]*domain.Room, error)
}

type ScheduleUsecase interface {
	CreateSchedule(ctx context.Context, roomID uuid.UUID, daysOfWeek []int, startTime, endTime string) (*domain.Schedule, error)
}

type SlotUsecase interface {
	GenerateAndGetAvailable(ctx context.Context, roomID uuid.UUID, date time.Time) ([]*domain.Slot, error)
}

type BookingUsecase interface {
	CreateBooking(ctx context.Context, userID, slotID uuid.UUID, createConferenceLink bool) (*domain.Booking, error)
	ListAll(ctx context.Context, page, pageSize int) ([]*domain.Booking, *domain.Pagination, error)
	ListMy(ctx context.Context, userID uuid.UUID) ([]*domain.Booking, error)
	CancelBooking(ctx context.Context, userID, bookingID uuid.UUID) (*domain.Booking, error)
}

type Deps struct {
	Auth      AuthUsecase
	Rooms     RoomUsecase
	Schedules ScheduleUsecase
	Slots     SlotUsecase
	Bookings  BookingUsecase
}

func NewRouter(deps Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	authMW := AuthMiddleware(deps.Auth)

	r.Get("/_info", handleInfo)
	r.Handle("/swagger/*", swaggerHandler())
	r.Post("/dummyLogin", handleDummyLogin(deps.Auth))
	r.Post("/register", handleRegister(deps.Auth))
	r.Post("/login", handleLogin(deps.Auth))

	r.Group(func(r chi.Router) {
		r.Use(authMW)

		r.Get("/rooms/list", handleListRooms(deps.Rooms))

		r.Group(func(r chi.Router) {
			r.Use(RequireRole("admin"))
			r.Post("/rooms/create", handleCreateRoom(deps.Rooms))
			r.Post("/rooms/{roomId}/schedule/create", handleCreateSchedule(deps.Schedules))
			r.Get("/bookings/list", handleListBookings(deps.Bookings))
		})

		r.Get("/rooms/{roomId}/slots/list", handleListSlots(deps.Slots))

		r.Group(func(r chi.Router) {
			r.Use(RequireRole("user"))
			r.Post("/bookings/create", handleCreateBooking(deps.Bookings))
			r.Get("/bookings/my", handleListMyBookings(deps.Bookings))
			r.Post("/bookings/{bookingId}/cancel", handleCancelBooking(deps.Bookings))
		})
	})

	return r
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]interface{}{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
