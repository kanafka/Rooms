package delivery_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"room-booking/internal/delivery"
	"room-booking/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockAuth struct{ mock.Mock }

func (m *mockAuth) DummyLogin(role string) (string, error) {
	args := m.Called(role)
	return args.String(0), args.Error(1)
}
func (m *mockAuth) Register(ctx context.Context, email, password, role string) (*domain.User, error) {
	args := m.Called(ctx, email, password, role)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *mockAuth) Login(ctx context.Context, email, password string) (string, error) {
	args := m.Called(ctx, email, password)
	return args.String(0), args.Error(1)
}
func (m *mockAuth) ValidateToken(tokenStr string) (*domain.Claims, error) {
	args := m.Called(tokenStr)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Claims), args.Error(1)
}

type mockRooms struct{ mock.Mock }

func (m *mockRooms) CreateRoom(ctx context.Context, name string, description *string, capacity *int) (*domain.Room, error) {
	args := m.Called(ctx, name, description, capacity)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Room), args.Error(1)
}
func (m *mockRooms) ListRooms(ctx context.Context) ([]*domain.Room, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Room), args.Error(1)
}

type mockSchedules struct{ mock.Mock }

func (m *mockSchedules) CreateSchedule(ctx context.Context, roomID uuid.UUID, daysOfWeek []int, startTime, endTime string) (*domain.Schedule, error) {
	args := m.Called(ctx, roomID, daysOfWeek, startTime, endTime)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Schedule), args.Error(1)
}

type mockSlots struct{ mock.Mock }

func (m *mockSlots) GenerateAndGetAvailable(ctx context.Context, roomID uuid.UUID, date time.Time) ([]*domain.Slot, error) {
	args := m.Called(ctx, roomID, date)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Slot), args.Error(1)
}

type mockBookings struct{ mock.Mock }

func (m *mockBookings) CreateBooking(ctx context.Context, userID, slotID uuid.UUID, createConferenceLink bool) (*domain.Booking, error) {
	args := m.Called(ctx, userID, slotID, createConferenceLink)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Booking), args.Error(1)
}
func (m *mockBookings) ListAll(ctx context.Context, page, pageSize int) ([]*domain.Booking, *domain.Pagination, error) {
	args := m.Called(ctx, page, pageSize)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).([]*domain.Booking), args.Get(1).(*domain.Pagination), args.Error(2)
}
func (m *mockBookings) ListMy(ctx context.Context, userID uuid.UUID) ([]*domain.Booking, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Booking), args.Error(1)
}
func (m *mockBookings) CancelBooking(ctx context.Context, userID, bookingID uuid.UUID) (*domain.Booking, error) {
	args := m.Called(ctx, userID, bookingID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Booking), args.Error(1)
}

func newRouter(deps delivery.Deps) http.Handler {
	return delivery.NewRouter(deps)
}

func defaultDeps() (delivery.Deps, *mockAuth, *mockRooms, *mockSchedules, *mockSlots, *mockBookings) {
	auth := new(mockAuth)
	rooms := new(mockRooms)
	schedules := new(mockSchedules)
	slots := new(mockSlots)
	bookings := new(mockBookings)
	deps := delivery.Deps{
		Auth:      auth,
		Rooms:     rooms,
		Schedules: schedules,
		Slots:     slots,
		Bookings:  bookings,
	}
	return deps, auth, rooms, schedules, slots, bookings
}

func tokenFor(auth *mockAuth, userID uuid.UUID, role string) string {
	token := "valid-token-" + role
	auth.On("ValidateToken", token).Return(&domain.Claims{UserID: userID, Role: role}, nil).Maybe()
	return "Bearer " + token
}

func jsonBody(v interface{}) *bytes.Buffer {
	b, _ := json.Marshal(v)
	return bytes.NewBuffer(b)
}

func TestHandleDummyLogin_Success(t *testing.T) {
	deps, auth, _, _, _, _ := defaultDeps()
	auth.On("DummyLogin", "admin").Return("tok123", nil)

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/dummyLogin", jsonBody(map[string]string{"role": "admin"}))

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "tok123", resp["token"])
}

func TestHandleDummyLogin_InvalidRole(t *testing.T) {
	deps, auth, _, _, _, _ := defaultDeps()
	auth.On("DummyLogin", "superadmin").Return("", domain.ErrInvalidRequest)

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/dummyLogin", jsonBody(map[string]string{"role": "superadmin"}))

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleDummyLogin_InvalidBody(t *testing.T) {
	deps, _, _, _, _, _ := defaultDeps()

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/dummyLogin", strings.NewReader("not-json"))

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleRegister_Success(t *testing.T) {
	deps, auth, _, _, _, _ := defaultDeps()
	user := &domain.User{ID: uuid.New(), Email: "a@b.com", Role: domain.RoleUser}
	auth.On("Register", mock.Anything, "a@b.com", "pass", "user").Return(user, nil)

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/register",
		jsonBody(map[string]string{"email": "a@b.com", "password": "pass", "role": "user"}))

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandleRegister_MissingFields(t *testing.T) {
	deps, _, _, _, _, _ := defaultDeps()

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/register",
		jsonBody(map[string]string{"email": "a@b.com"}))

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleRegister_EmailTaken(t *testing.T) {
	deps, auth, _, _, _, _ := defaultDeps()
	auth.On("Register", mock.Anything, "a@b.com", "pass", "user").Return(nil, domain.ErrEmailTaken)

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/register",
		jsonBody(map[string]string{"email": "a@b.com", "password": "pass", "role": "user"}))

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "INVALID_REQUEST")
}

func TestHandleRegister_InvalidBody(t *testing.T) {
	deps, _, _, _, _, _ := defaultDeps()

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader("bad"))

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleLogin_Success(t *testing.T) {
	deps, auth, _, _, _, _ := defaultDeps()
	auth.On("Login", mock.Anything, "a@b.com", "pass").Return("jwt", nil)

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/login",
		jsonBody(map[string]string{"email": "a@b.com", "password": "pass"}))

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "jwt", resp["token"])
}

func TestHandleLogin_InvalidCredentials(t *testing.T) {
	deps, auth, _, _, _, _ := defaultDeps()
	auth.On("Login", mock.Anything, "a@b.com", "wrong").Return("", domain.ErrInvalidCredentials)

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/login",
		jsonBody(map[string]string{"email": "a@b.com", "password": "wrong"}))

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleLogin_MissingFields(t *testing.T) {
	deps, _, _, _, _, _ := defaultDeps()

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/login",
		jsonBody(map[string]string{"email": "a@b.com"}))

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	deps, _, rooms, _, _, _ := defaultDeps()
	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/rooms/list", nil)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	rooms.AssertNotCalled(t, "ListRooms")
}

func TestAuthMiddleware_InvalidFormat(t *testing.T) {
	deps, _, _, _, _, _ := defaultDeps()
	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/rooms/list", nil)
	req.Header.Set("Authorization", "InvalidHeader")

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	deps, auth, _, _, _, _ := defaultDeps()
	auth.On("ValidateToken", "badtoken").Return(nil, domain.ErrUnauthorized)

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/rooms/list", nil)
	req.Header.Set("Authorization", "Bearer badtoken")

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleListRooms_Success(t *testing.T) {
	deps, auth, rooms, _, _, _ := defaultDeps()
	userID := uuid.New()
	authHeader := tokenFor(auth, userID, "user")
	rooms.On("ListRooms", mock.Anything).Return([]*domain.Room{
		{ID: uuid.New(), Name: "Room A"},
	}, nil)

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/rooms/list", nil)
	req.Header.Set("Authorization", authHeader)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Len(t, resp["rooms"], 1)
}

func TestHandleListRooms_Empty(t *testing.T) {
	deps, auth, rooms, _, _, _ := defaultDeps()
	userID := uuid.New()
	authHeader := tokenFor(auth, userID, "user")
	rooms.On("ListRooms", mock.Anything).Return(nil, nil)

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/rooms/list", nil)
	req.Header.Set("Authorization", authHeader)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.NotNil(t, resp["rooms"])
}

func TestHandleListRooms_Error(t *testing.T) {
	deps, auth, rooms, _, _, _ := defaultDeps()
	userID := uuid.New()
	authHeader := tokenFor(auth, userID, "user")
	rooms.On("ListRooms", mock.Anything).Return(nil, fmt.Errorf("db error"))

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/rooms/list", nil)
	req.Header.Set("Authorization", authHeader)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandleCreateRoom_Success(t *testing.T) {
	deps, auth, rooms, _, _, _ := defaultDeps()
	adminID := uuid.New()
	authHeader := tokenFor(auth, adminID, "admin")
	room := &domain.Room{ID: uuid.New(), Name: "Big Room"}
	rooms.On("CreateRoom", mock.Anything, "Big Room", (*string)(nil), (*int)(nil)).Return(room, nil)

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rooms/create",
		jsonBody(map[string]interface{}{"name": "Big Room"}))
	req.Header.Set("Authorization", authHeader)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandleCreateRoom_MissingName(t *testing.T) {
	deps, auth, _, _, _, _ := defaultDeps()
	adminID := uuid.New()
	authHeader := tokenFor(auth, adminID, "admin")

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rooms/create",
		jsonBody(map[string]interface{}{}))
	req.Header.Set("Authorization", authHeader)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleCreateRoom_ForbiddenForUser(t *testing.T) {
	deps, auth, _, _, _, _ := defaultDeps()
	userID := uuid.New()
	authHeader := tokenFor(auth, userID, "user")

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rooms/create",
		jsonBody(map[string]interface{}{"name": "Room"}))
	req.Header.Set("Authorization", authHeader)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandleCreateSchedule_Success(t *testing.T) {
	deps, auth, _, schedules, _, _ := defaultDeps()
	adminID := uuid.New()
	authHeader := tokenFor(auth, adminID, "admin")
	roomID := uuid.New()
	sched := &domain.Schedule{ID: uuid.New(), RoomID: roomID}
	schedules.On("CreateSchedule", mock.Anything, roomID, []int{1, 2}, "09:00", "18:00").Return(sched, nil)

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rooms/"+roomID.String()+"/schedule/create",
		jsonBody(map[string]interface{}{"daysOfWeek": []int{1, 2}, "startTime": "09:00", "endTime": "18:00"}))
	req.Header.Set("Authorization", authHeader)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandleCreateSchedule_InvalidRoomID(t *testing.T) {
	deps, auth, _, _, _, _ := defaultDeps()
	adminID := uuid.New()
	authHeader := tokenFor(auth, adminID, "admin")

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rooms/not-a-uuid/schedule/create",
		jsonBody(map[string]interface{}{"daysOfWeek": []int{1}, "startTime": "09:00", "endTime": "18:00"}))
	req.Header.Set("Authorization", authHeader)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleCreateSchedule_MissingFields(t *testing.T) {
	deps, auth, _, _, _, _ := defaultDeps()
	adminID := uuid.New()
	authHeader := tokenFor(auth, adminID, "admin")
	roomID := uuid.New()

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rooms/"+roomID.String()+"/schedule/create",
		jsonBody(map[string]interface{}{"daysOfWeek": []int{}}))
	req.Header.Set("Authorization", authHeader)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleCreateSchedule_RoomNotFound(t *testing.T) {
	deps, auth, _, schedules, _, _ := defaultDeps()
	adminID := uuid.New()
	authHeader := tokenFor(auth, adminID, "admin")
	roomID := uuid.New()
	schedules.On("CreateSchedule", mock.Anything, roomID, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, domain.ErrRoomNotFound)

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rooms/"+roomID.String()+"/schedule/create",
		jsonBody(map[string]interface{}{"daysOfWeek": []int{1}, "startTime": "09:00", "endTime": "18:00"}))
	req.Header.Set("Authorization", authHeader)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleCreateSchedule_AlreadyExists(t *testing.T) {
	deps, auth, _, schedules, _, _ := defaultDeps()
	adminID := uuid.New()
	authHeader := tokenFor(auth, adminID, "admin")
	roomID := uuid.New()
	schedules.On("CreateSchedule", mock.Anything, roomID, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, domain.ErrScheduleExists)

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rooms/"+roomID.String()+"/schedule/create",
		jsonBody(map[string]interface{}{"daysOfWeek": []int{1}, "startTime": "09:00", "endTime": "18:00"}))
	req.Header.Set("Authorization", authHeader)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestHandleListSlots_Success(t *testing.T) {
	deps, auth, _, _, slots, _ := defaultDeps()
	userID := uuid.New()
	authHeader := tokenFor(auth, userID, "user")
	roomID := uuid.New()
	date, _ := time.Parse("2006-01-02", "2026-05-01")
	slot := &domain.Slot{ID: uuid.New(), RoomID: roomID, Start: date, End: date.Add(30 * time.Minute)}
	slots.On("GenerateAndGetAvailable", mock.Anything, roomID, date.UTC()).Return([]*domain.Slot{slot}, nil)

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/rooms/"+roomID.String()+"/slots/list?date=2026-05-01", nil)
	req.Header.Set("Authorization", authHeader)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleListSlots_InvalidRoomID(t *testing.T) {
	deps, auth, _, _, _, _ := defaultDeps()
	userID := uuid.New()
	authHeader := tokenFor(auth, userID, "user")

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/rooms/bad-id/slots/list?date=2026-05-01", nil)
	req.Header.Set("Authorization", authHeader)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleListSlots_MissingDate(t *testing.T) {
	deps, auth, _, _, _, _ := defaultDeps()
	userID := uuid.New()
	authHeader := tokenFor(auth, userID, "user")
	roomID := uuid.New()

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/rooms/"+roomID.String()+"/slots/list", nil)
	req.Header.Set("Authorization", authHeader)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleListSlots_BadDateFormat(t *testing.T) {
	deps, auth, _, _, _, _ := defaultDeps()
	userID := uuid.New()
	authHeader := tokenFor(auth, userID, "user")
	roomID := uuid.New()

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/rooms/"+roomID.String()+"/slots/list?date=01-05-2026", nil)
	req.Header.Set("Authorization", authHeader)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleListSlots_RoomNotFound(t *testing.T) {
	deps, auth, _, _, slots, _ := defaultDeps()
	userID := uuid.New()
	authHeader := tokenFor(auth, userID, "user")
	roomID := uuid.New()
	slots.On("GenerateAndGetAvailable", mock.Anything, roomID, mock.Anything).Return(nil, domain.ErrRoomNotFound)

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/rooms/"+roomID.String()+"/slots/list?date=2026-05-01", nil)
	req.Header.Set("Authorization", authHeader)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleListSlots_EmptyResult(t *testing.T) {
	deps, auth, _, _, slots, _ := defaultDeps()
	userID := uuid.New()
	authHeader := tokenFor(auth, userID, "user")
	roomID := uuid.New()
	slots.On("GenerateAndGetAvailable", mock.Anything, roomID, mock.Anything).Return(nil, nil)

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/rooms/"+roomID.String()+"/slots/list?date=2026-05-01", nil)
	req.Header.Set("Authorization", authHeader)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.NotNil(t, resp["slots"])
}

func TestHandleCreateBooking_Success(t *testing.T) {
	deps, auth, _, _, _, bookings := defaultDeps()
	userID := uuid.New()
	authHeader := tokenFor(auth, userID, "user")
	slotID := uuid.New()
	booking := &domain.Booking{ID: uuid.New(), SlotID: slotID, UserID: userID, Status: domain.BookingStatusActive}
	bookings.On("CreateBooking", mock.Anything, userID, slotID, false).Return(booking, nil)

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/bookings/create",
		jsonBody(map[string]interface{}{"slotId": slotID.String(), "createConferenceLink": false}))
	req.Header.Set("Authorization", authHeader)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandleCreateBooking_InvalidBody(t *testing.T) {
	deps, auth, _, _, _, _ := defaultDeps()
	userID := uuid.New()
	authHeader := tokenFor(auth, userID, "user")

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/bookings/create", strings.NewReader("bad"))
	req.Header.Set("Authorization", authHeader)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleCreateBooking_InvalidSlotID(t *testing.T) {
	deps, auth, _, _, _, _ := defaultDeps()
	userID := uuid.New()
	authHeader := tokenFor(auth, userID, "user")

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/bookings/create",
		jsonBody(map[string]interface{}{"slotId": "not-uuid"}))
	req.Header.Set("Authorization", authHeader)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleCreateBooking_SlotNotFound(t *testing.T) {
	deps, auth, _, _, _, bookings := defaultDeps()
	userID := uuid.New()
	authHeader := tokenFor(auth, userID, "user")
	slotID := uuid.New()
	bookings.On("CreateBooking", mock.Anything, userID, slotID, false).Return(nil, domain.ErrSlotNotFound)

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/bookings/create",
		jsonBody(map[string]interface{}{"slotId": slotID.String()}))
	req.Header.Set("Authorization", authHeader)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleCreateBooking_SlotAlreadyBooked(t *testing.T) {
	deps, auth, _, _, _, bookings := defaultDeps()
	userID := uuid.New()
	authHeader := tokenFor(auth, userID, "user")
	slotID := uuid.New()
	bookings.On("CreateBooking", mock.Anything, userID, slotID, false).Return(nil, domain.ErrSlotBooked)

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/bookings/create",
		jsonBody(map[string]interface{}{"slotId": slotID.String()}))
	req.Header.Set("Authorization", authHeader)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestHandleCreateBooking_ForbiddenForAdmin(t *testing.T) {
	deps, auth, _, _, _, _ := defaultDeps()
	adminID := uuid.New()
	authHeader := tokenFor(auth, adminID, "admin")

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/bookings/create",
		jsonBody(map[string]interface{}{"slotId": uuid.New().String()}))
	req.Header.Set("Authorization", authHeader)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandleListBookings_Success(t *testing.T) {
	deps, auth, _, _, _, bookings := defaultDeps()
	adminID := uuid.New()
	authHeader := tokenFor(auth, adminID, "admin")
	pagination := &domain.Pagination{Page: 1, PageSize: 20, Total: 1}
	bookings.On("ListAll", mock.Anything, 1, 20).Return([]*domain.Booking{{ID: uuid.New()}}, pagination, nil)

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/bookings/list", nil)
	req.Header.Set("Authorization", authHeader)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.NotNil(t, resp["bookings"])
	assert.NotNil(t, resp["pagination"])
}

func TestHandleListBookings_WithPaginationParams(t *testing.T) {
	deps, auth, _, _, _, bookings := defaultDeps()
	adminID := uuid.New()
	authHeader := tokenFor(auth, adminID, "admin")
	pagination := &domain.Pagination{Page: 2, PageSize: 10, Total: 5}
	bookings.On("ListAll", mock.Anything, 2, 10).Return([]*domain.Booking{}, pagination, nil)

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/bookings/list?page=2&pageSize=10", nil)
	req.Header.Set("Authorization", authHeader)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleListBookings_PageSizeExceedsMax(t *testing.T) {
	deps, auth, _, _, _, bookings := defaultDeps()
	adminID := uuid.New()
	authHeader := tokenFor(auth, adminID, "admin")
	pagination := &domain.Pagination{Page: 1, PageSize: 100, Total: 0}
	bookings.On("ListAll", mock.Anything, 1, 100).Return([]*domain.Booking{}, pagination, nil)

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/bookings/list?pageSize=500", nil)
	req.Header.Set("Authorization", authHeader)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	bookings.AssertCalled(t, "ListAll", mock.Anything, 1, 100)
}

func TestHandleListMyBookings_Success(t *testing.T) {
	deps, auth, _, _, _, bookings := defaultDeps()
	userID := uuid.New()
	authHeader := tokenFor(auth, userID, "user")
	bookings.On("ListMy", mock.Anything, userID).Return([]*domain.Booking{
		{ID: uuid.New(), UserID: userID, Status: domain.BookingStatusActive},
	}, nil)

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/bookings/my", nil)
	req.Header.Set("Authorization", authHeader)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Len(t, resp["bookings"], 1)
}

func TestHandleListMyBookings_Error(t *testing.T) {
	deps, auth, _, _, _, bookings := defaultDeps()
	userID := uuid.New()
	authHeader := tokenFor(auth, userID, "user")
	bookings.On("ListMy", mock.Anything, userID).Return(nil, fmt.Errorf("db error"))

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/bookings/my", nil)
	req.Header.Set("Authorization", authHeader)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandleCancelBooking_Success(t *testing.T) {
	deps, auth, _, _, _, bookings := defaultDeps()
	userID := uuid.New()
	authHeader := tokenFor(auth, userID, "user")
	bookingID := uuid.New()
	booking := &domain.Booking{ID: bookingID, UserID: userID, Status: domain.BookingStatusCancelled}
	bookings.On("CancelBooking", mock.Anything, userID, bookingID).Return(booking, nil)

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/bookings/"+bookingID.String()+"/cancel", nil)
	req.Header.Set("Authorization", authHeader)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleCancelBooking_InvalidID(t *testing.T) {
	deps, auth, _, _, _, _ := defaultDeps()
	userID := uuid.New()
	authHeader := tokenFor(auth, userID, "user")

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/bookings/not-uuid/cancel", nil)
	req.Header.Set("Authorization", authHeader)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleCancelBooking_NotFound(t *testing.T) {
	deps, auth, _, _, _, bookings := defaultDeps()
	userID := uuid.New()
	authHeader := tokenFor(auth, userID, "user")
	bookingID := uuid.New()
	bookings.On("CancelBooking", mock.Anything, userID, bookingID).Return(nil, domain.ErrBookingNotFound)

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/bookings/"+bookingID.String()+"/cancel", nil)
	req.Header.Set("Authorization", authHeader)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleCancelBooking_Forbidden(t *testing.T) {
	deps, auth, _, _, _, bookings := defaultDeps()
	userID := uuid.New()
	authHeader := tokenFor(auth, userID, "user")
	bookingID := uuid.New()
	bookings.On("CancelBooking", mock.Anything, userID, bookingID).Return(nil, domain.ErrForbidden)

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/bookings/"+bookingID.String()+"/cancel", nil)
	req.Header.Set("Authorization", authHeader)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandleInfo(t *testing.T) {
	deps, _, _, _, _, _ := defaultDeps()

	r := newRouter(deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/_info", nil)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestChiRouting_RoomIDParam(t *testing.T) {
	r := chi.NewRouter()
	called := false
	r.Get("/rooms/{roomId}/slots/list", func(w http.ResponseWriter, req *http.Request) {
		called = true
		assert.Equal(t, "abc123", chi.URLParam(req, "roomId"))
		w.WriteHeader(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/rooms/abc123/slots/list", nil)
	r.ServeHTTP(w, req)

	assert.True(t, called)
}
