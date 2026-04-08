//go:build e2e

package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"room-booking/internal/config"
	"room-booking/internal/delivery"
	"room-booking/internal/repository/postgres"
	"room-booking/internal/usecase"
)

func setupTestServer(t *testing.T) (*httptest.Server, func()) {
	t.Helper()

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping e2e tests")
	}

	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, dbURL)
	require.NoError(t, err)

	migrationSQL, err := os.ReadFile("../../migrations/001_init.sql")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, string(migrationSQL))
	require.NoError(t, err)

	cfg := config.Load()
	cfg.DatabaseURL = dbURL

	adminUUID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	userUUID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	userRepo := postgres.NewUserRepo(pool)
	roomRepo := postgres.NewRoomRepo(pool)
	scheduleRepo := postgres.NewScheduleRepo(pool)
	slotRepo := postgres.NewSlotRepo(pool)
	bookingRepo := postgres.NewBookingRepo(pool)

	authUC := usecase.NewAuthUsecase(userRepo, cfg.JWTSecret, adminUUID, userUUID)
	roomUC := usecase.NewRoomUsecase(roomRepo)
	scheduleUC := usecase.NewScheduleUsecase(scheduleRepo, roomRepo)
	slotUC := usecase.NewSlotUsecase(slotRepo, roomRepo, scheduleRepo)
	conferenceUC := &usecase.MockConferenceService{}
	bookingUC := usecase.NewBookingUsecase(bookingRepo, slotRepo, conferenceUC)

	deps := delivery.Deps{
		Auth:      authUC,
		Rooms:     roomUC,
		Schedules: scheduleUC,
		Slots:     slotUC,
		Bookings:  bookingUC,
	}

	router := delivery.NewRouter(deps)
	server := httptest.NewServer(router)

	cleanup := func() {
		server.Close()
		_, _ = pool.Exec(ctx, "DELETE FROM bookings")
		_, _ = pool.Exec(ctx, "DELETE FROM slots")
		_, _ = pool.Exec(ctx, "DELETE FROM schedules")
		_, _ = pool.Exec(ctx, "DELETE FROM rooms")
		_, _ = pool.Exec(ctx, "DELETE FROM users")
		pool.Close()
	}

	return server, cleanup
}

func doRequest(t *testing.T, server *httptest.Server, method, path string, body interface{}, token string) *http.Response {
	t.Helper()

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		require.NoError(t, err)
	}

	req, err := http.NewRequest(method, server.URL+path, bytes.NewBuffer(bodyBytes))
	require.NoError(t, err)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	return resp
}

func getToken(t *testing.T, server *httptest.Server, role string) string {
	t.Helper()

	resp := doRequest(t, server, http.MethodPost, "/dummyLogin", map[string]string{"role": role}, "")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	return result["token"]
}

func TestCreateRoomScheduleBooking(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	adminToken := getToken(t, server, "admin")
	userToken := getToken(t, server, "user")

	resp := doRequest(t, server, http.MethodPost, "/rooms/create",
		map[string]interface{}{
			"name":        "E2E Test Room",
			"description": "Test room for e2e",
			"capacity":    10,
		},
		adminToken,
	)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var roomResp map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&roomResp))
	roomData := roomResp["room"].(map[string]interface{})
	roomID := roomData["id"].(string)

	resp2 := doRequest(t, server, http.MethodPost, fmt.Sprintf("/rooms/%s/schedule/create", roomID),
		map[string]interface{}{
			"daysOfWeek": []int{1, 2, 3, 4, 5, 6, 7},
			"startTime":  "09:00",
			"endTime":    "10:00",
		},
		adminToken,
	)
	defer resp2.Body.Close()
	require.Equal(t, http.StatusCreated, resp2.StatusCode)

	tomorrow := time.Now().Add(24 * time.Hour).Format("2006-01-02")
	resp3 := doRequest(t, server, http.MethodGet,
		fmt.Sprintf("/rooms/%s/slots/list?date=%s", roomID, tomorrow),
		nil, userToken,
	)
	defer resp3.Body.Close()
	require.Equal(t, http.StatusOK, resp3.StatusCode)

	var slotsResp map[string]interface{}
	require.NoError(t, json.NewDecoder(resp3.Body).Decode(&slotsResp))
	slotsData := slotsResp["slots"].([]interface{})
	require.NotEmpty(t, slotsData)

	slotData := slotsData[0].(map[string]interface{})
	slotID := slotData["id"].(string)

	resp4 := doRequest(t, server, http.MethodPost, "/bookings/create",
		map[string]interface{}{
			"slotId":               slotID,
			"createConferenceLink": false,
		},
		userToken,
	)
	defer resp4.Body.Close()
	require.Equal(t, http.StatusCreated, resp4.StatusCode)

	var bookingResp map[string]interface{}
	require.NoError(t, json.NewDecoder(resp4.Body).Decode(&bookingResp))
	bookingData := bookingResp["booking"].(map[string]interface{})
	assert.Equal(t, "active", bookingData["status"])
}

func TestCancelBooking(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	adminToken := getToken(t, server, "admin")
	userToken := getToken(t, server, "user")

	resp := doRequest(t, server, http.MethodPost, "/rooms/create",
		map[string]interface{}{
			"name": "Cancel Test Room",
		},
		adminToken,
	)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var roomResp map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&roomResp))
	roomData := roomResp["room"].(map[string]interface{})
	roomID := roomData["id"].(string)

	resp2 := doRequest(t, server, http.MethodPost, fmt.Sprintf("/rooms/%s/schedule/create", roomID),
		map[string]interface{}{
			"daysOfWeek": []int{1, 2, 3, 4, 5, 6, 7},
			"startTime":  "14:00",
			"endTime":    "15:00",
		},
		adminToken,
	)
	defer resp2.Body.Close()
	require.Equal(t, http.StatusCreated, resp2.StatusCode)

	tomorrow := time.Now().Add(24 * time.Hour).Format("2006-01-02")
	resp3 := doRequest(t, server, http.MethodGet,
		fmt.Sprintf("/rooms/%s/slots/list?date=%s", roomID, tomorrow),
		nil, userToken,
	)
	defer resp3.Body.Close()
	require.Equal(t, http.StatusOK, resp3.StatusCode)

	var slotsResp map[string]interface{}
	require.NoError(t, json.NewDecoder(resp3.Body).Decode(&slotsResp))
	slotsData := slotsResp["slots"].([]interface{})
	require.NotEmpty(t, slotsData)

	slotData := slotsData[0].(map[string]interface{})
	slotID := slotData["id"].(string)

	resp4 := doRequest(t, server, http.MethodPost, "/bookings/create",
		map[string]interface{}{
			"slotId": slotID,
		},
		userToken,
	)
	defer resp4.Body.Close()
	require.Equal(t, http.StatusCreated, resp4.StatusCode)

	var bookingResp map[string]interface{}
	require.NoError(t, json.NewDecoder(resp4.Body).Decode(&bookingResp))
	bookingData := bookingResp["booking"].(map[string]interface{})
	bookingID := bookingData["id"].(string)

	resp5 := doRequest(t, server, http.MethodPost, fmt.Sprintf("/bookings/%s/cancel", bookingID),
		nil, userToken,
	)
	defer resp5.Body.Close()
	require.Equal(t, http.StatusOK, resp5.StatusCode)

	var cancelResp map[string]interface{}
	require.NoError(t, json.NewDecoder(resp5.Body).Decode(&cancelResp))
	cancelledBooking := cancelResp["booking"].(map[string]interface{})
	assert.Equal(t, "cancelled", cancelledBooking["status"])
}
