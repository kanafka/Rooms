// @title           Room Booking Service
// @version         1.0
// @description     Сервис бронирования переговорок. Для авторизации используйте /dummyLogin, скопируйте токен и вставьте в поле Authorize.
// @host            localhost:8080
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
// @description     Введите токен в формате: Bearer {token}
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "room-booking/docs"
	"room-booking/internal/config"
	"room-booking/internal/delivery"
	"room-booking/internal/repository/postgres"
	"room-booking/internal/usecase"
)

func main() {
	cfg := config.Load()

	ctx := context.Background()

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	if err := runMigrations(ctx, pool); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	userRepo := postgres.NewUserRepo(pool)
	roomRepo := postgres.NewRoomRepo(pool)
	scheduleRepo := postgres.NewScheduleRepo(pool)
	slotRepo := postgres.NewSlotRepo(pool)
	bookingRepo := postgres.NewBookingRepo(pool)

	authUC := usecase.NewAuthUsecase(userRepo, cfg.JWTSecret, cfg.AdminUUID, cfg.UserUUID)
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

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("starting server on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func runMigrations(ctx context.Context, pool *postgres.Pool) error {
	migrationSQL, err := os.ReadFile("migrations/001_init.sql")
	if err != nil {
		return fmt.Errorf("read migration file: %w", err)
	}

	_, err = pool.Exec(ctx, string(migrationSQL))
	if err != nil {
		return fmt.Errorf("execute migration: %w", err)
	}

	log.Println("migrations applied successfully")
	return nil
}
