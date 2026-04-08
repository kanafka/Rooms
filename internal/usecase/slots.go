package usecase

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	"room-booking/internal/domain"
)

type SlotUsecase struct {
	slots     domain.SlotRepository
	rooms     domain.RoomRepository
	schedules domain.ScheduleRepository
}

func NewSlotUsecase(slots domain.SlotRepository, rooms domain.RoomRepository, schedules domain.ScheduleRepository) *SlotUsecase {
	return &SlotUsecase{
		slots:     slots,
		rooms:     rooms,
		schedules: schedules,
	}
}

func (u *SlotUsecase) GenerateAndGetAvailable(ctx context.Context, roomID uuid.UUID, date time.Time) ([]*domain.Slot, error) {
	if _, err := u.rooms.GetByID(ctx, roomID); err != nil {
		return nil, err
	}

	schedule, err := u.schedules.GetByRoomID(ctx, roomID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return []*domain.Slot{}, nil
		}
		return nil, err
	}

	wd := date.UTC().Weekday()
	var isoDay int
	if wd == time.Sunday {
		isoDay = 7
	} else {
		isoDay = int(wd)
	}

	inSchedule := false
	for _, d := range schedule.DaysOfWeek {
		if d == isoDay {
			inSchedule = true
			break
		}
	}
	if !inSchedule {
		return []*domain.Slot{}, nil
	}

	slots, err := generateSlots(roomID, date.UTC(), schedule.StartTime, schedule.EndTime)
	if err != nil {
		return nil, err
	}

	if err := u.slots.UpsertSlots(ctx, slots); err != nil {
		return nil, err
	}

	return u.slots.GetAvailableByRoomAndDate(ctx, roomID, date)
}

func generateSlots(roomID uuid.UUID, date time.Time, startTime, endTime string) ([]*domain.Slot, error) {
	startH, startM, err := parseHHMM(startTime)
	if err != nil {
		return nil, err
	}
	endH, endM, err := parseHHMM(endTime)
	if err != nil {
		return nil, err
	}

	dayStart := time.Date(date.Year(), date.Month(), date.Day(), startH, startM, 0, 0, time.UTC)
	dayEnd := time.Date(date.Year(), date.Month(), date.Day(), endH, endM, 0, 0, time.UTC)

	var slots []*domain.Slot
	slotDuration := 30 * time.Minute

	for cur := dayStart; cur.Add(slotDuration).Before(dayEnd) || cur.Add(slotDuration).Equal(dayEnd); cur = cur.Add(slotDuration) {
		slotEnd := cur.Add(slotDuration)
		slotID := uuid.NewSHA1(uuid.NameSpaceURL, []byte(roomID.String()+"|"+cur.UTC().Format(time.RFC3339)))
		slots = append(slots, &domain.Slot{
			ID:     slotID,
			RoomID: roomID,
			Start:  cur,
			End:    slotEnd,
		})
	}

	return slots, nil
}

func parseHHMM(t string) (int, int, error) {
	h, err := strconv.Atoi(t[0:2])
	if err != nil {
		return 0, 0, err
	}
	m, err := strconv.Atoi(t[3:5])
	if err != nil {
		return 0, 0, err
	}
	return h, m, nil
}
