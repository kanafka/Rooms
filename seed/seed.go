package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/booking?sslmode=disable"
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("ping database: %v", err)
	}

	log.Println("connected to database, seeding...")

	rooms := []struct {
		name        string
		description string
		capacity    int
	}{
		{"Main Conference Room", "Large conference room with projector", 20},
		{"Small Meeting Room", "Cozy room for small meetings", 6},
		{"Board Room", "Executive board room with video conferencing", 12},
	}

	for _, r := range rooms {
		roomID := uuid.New()
		_, err := pool.Exec(ctx,
			`INSERT INTO rooms (id, name, description, capacity, created_at)
			 VALUES ($1, $2, $3, $4, $5)
			 ON CONFLICT DO NOTHING`,
			roomID, r.name, r.description, r.capacity, time.Now().UTC(),
		)
		if err != nil {
			log.Printf("insert room %s: %v", r.name, err)
			continue
		}
		log.Printf("created room: %s (id: %s)", r.name, roomID)

		scheduleID := uuid.New()
		_, err = pool.Exec(ctx,
			`INSERT INTO schedules (id, room_id, days_of_week, start_time, end_time)
			 VALUES ($1, $2, $3, $4, $5)
			 ON CONFLICT (room_id) DO NOTHING`,
			scheduleID, roomID, []int{1, 2, 3, 4, 5}, "09:00", "18:00",
		)
		if err != nil {
			log.Printf("insert schedule for room %s: %v", r.name, err)
			continue
		}
		log.Printf("created schedule for room: %s (Mon-Fri 09:00-18:00)", r.name)
	}

	fmt.Println("seeding completed successfully")
}
