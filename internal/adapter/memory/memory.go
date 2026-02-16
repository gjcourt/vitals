// Package memory implements an in-memory repository for development and testing.
package memory

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"biometrics/internal/domain"
)

// DB implements an in-memory database storage.
type DB struct {
	mu          sync.Mutex
	weights     []domain.WeightEntry
	waterEvents []domain.WaterEvent
	users       []*domain.User
	sessions    map[string]*domain.Session

	weightIDCounter int64
	waterIDCounter  int64
	userIDCounter   int64
}

// New creates a new in-memory database.
func New() *DB {
	return &DB{
		sessions: make(map[string]*domain.Session),
	}
}

// Ensure interfaces are met.
var _ domain.WeightRepository = (*DB)(nil)
var _ domain.WaterRepository = (*DB)(nil)
var _ domain.UserRepository = (*DB)(nil)
var _ domain.SessionRepository = (*SessionRepo)(nil)

// --- WeightRepository ---

// AddWeightEvent adds a weight event.
func (db *DB) AddWeightEvent(ctx context.Context, value float64, unit string, createdAt time.Time) (int64, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.weightIDCounter++
	id := db.weightIDCounter

	entry := domain.WeightEntry{
		ID:        id,
		Value:     value,
		Unit:      unit,
		CreatedAt: createdAt.UTC(),
	}
	db.weights = append(db.weights, entry)
	return id, nil
}

// DeleteLatestWeightEvent deletes the most recent weight event.
func (db *DB) DeleteLatestWeightEvent(ctx context.Context) (bool, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if len(db.weights) == 0 {
		return false, nil
	}

	// Find index of latest created_at
	lastIdx := -1
	var lastTime time.Time

	for i, w := range db.weights {
		if lastIdx == -1 || w.CreatedAt.After(lastTime) {
			lastIdx = i
			lastTime = w.CreatedAt
		}
	}

	if lastIdx != -1 {
		// remove element
		db.weights = append(db.weights[:lastIdx], db.weights[lastIdx+1:]...)
		return true, nil
	}
	return false, nil
}

// LatestWeightForLocalDay returns the latest weight for the given day.
func (db *DB) LatestWeightForLocalDay(ctx context.Context, localDay string) (*domain.WeightEntry, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	dayStart, err := time.ParseInLocation("2006-01-02", localDay, time.Local)
	if err != nil {
		return nil, err
	}
	dayEnd := dayStart.Add(24 * time.Hour)

	var latest *domain.WeightEntry

	for i := range db.weights {
		w := &db.weights[i]
		// Compare using UTC as that's how it's stored and Postgres does comparison
		if !w.CreatedAt.Before(dayStart.UTC()) && w.CreatedAt.Before(dayEnd.UTC()) {
			if latest == nil || w.CreatedAt.After(latest.CreatedAt) {
				latest = w
			}
		}
	}

	if latest != nil {
		// we return a copy with Day set
		ret := *latest
		ret.Day = localDay
		return &ret, nil
	}
	return nil, nil
}

// ListRecentWeightEvents lists the most recent weight events.
func (db *DB) ListRecentWeightEvents(ctx context.Context, limit int) ([]domain.WeightEntry, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	// copy
	result := make([]domain.WeightEntry, len(db.weights))
	copy(result, db.weights)

	// sort desc
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	if len(result) > limit {
		result = result[:limit]
	}

	// Populate Day field based on CreatedAt in Local time
	for i := range result {
		result[i].Day = result[i].CreatedAt.In(time.Local).Format("2006-01-02")
	}

	return result, nil
}

// --- WaterRepository ---

// AddWaterEvent adds a water event.
func (db *DB) AddWaterEvent(ctx context.Context, deltaLiters float64, createdAt time.Time) (int64, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.waterIDCounter++
	id := db.waterIDCounter

	event := domain.WaterEvent{
		ID:          id,
		DeltaLiters: deltaLiters,
		CreatedAt:   createdAt.UTC(),
	}
	db.waterEvents = append(db.waterEvents, event)
	return id, nil
}

// DeleteWaterEvent deletes a water event by ID.
func (db *DB) DeleteWaterEvent(ctx context.Context, id int64) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	for i, w := range db.waterEvents {
		if w.ID == id {
			db.waterEvents = append(db.waterEvents[:i], db.waterEvents[i+1:]...)
			return nil
		}
	}
	// No error if not found, usually?
	return nil
}

// ListRecentWaterEvents lists the most recent water events.
func (db *DB) ListRecentWaterEvents(ctx context.Context, limit int) ([]domain.WaterEvent, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	result := make([]domain.WaterEvent, len(db.waterEvents))
	copy(result, db.waterEvents)

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

// WaterTotalForLocalDay returns the total water intake for the given day.
func (db *DB) WaterTotalForLocalDay(ctx context.Context, localDay string) (float64, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	dayStart, err := time.ParseInLocation("2006-01-02", localDay, time.Local)
	if err != nil {
		return 0, err
	}
	dayEnd := dayStart.Add(24 * time.Hour)

	var total float64
	for _, w := range db.waterEvents {
		if !w.CreatedAt.Before(dayStart.UTC()) && w.CreatedAt.Before(dayEnd.UTC()) {
			total += w.DeltaLiters
		}
	}
	return total, nil
}

// --- UserRepository ---

// GetByUsername retrieves a user by username.
func (db *DB) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, u := range db.users {
		if u.Username == username {
			return u, nil
		}
	}
	// Return nil if not found
	return nil, nil
}

// GetByID retrieves a user by ID.
func (db *DB) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, u := range db.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, nil
}

// Create creates a new user.
func (db *DB) Create(ctx context.Context, username, passwordHash string) (*domain.User, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, u := range db.users {
		if u.Username == username {
			return nil, errors.New("user already exists")
		}
	}

	db.userIDCounter++
	u := &domain.User{
		ID:           db.userIDCounter,
		Username:     username,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now().UTC(),
	}
	db.users = append(db.users, u)
	return u, nil
}

// Count returns the total number of users.
func (db *DB) Count(ctx context.Context) (int, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	return len(db.users), nil
}

// --- SessionRepository ---

// SessionRepo implements session persistence.
type SessionRepo struct {
	db *DB
}

// NewSessionRepo creates a new session repository.
func (db *DB) NewSessionRepo() *SessionRepo {
	return &SessionRepo{db: db}
}

// Create creates a new session.
func (r *SessionRepo) Create(ctx context.Context, userID int64, token string, expiresAt time.Time) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	r.db.sessions[token] = &domain.Session{
		Token:     token,
		UserID:    userID,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now().UTC(),
	}
	return nil
}

// GetByToken retrieves a session by token.
func (r *SessionRepo) GetByToken(ctx context.Context, token string) (*domain.Session, error) {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	if s, ok := r.db.sessions[token]; ok {
		if time.Now().After(s.ExpiresAt) {
			delete(r.db.sessions, token)
			return nil, nil
		}
		return s, nil
	}
	return nil, nil
}

// Delete deletes a session.
func (r *SessionRepo) Delete(ctx context.Context, token string) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()
	delete(r.db.sessions, token)
	return nil
}

// DeleteExpired deletes all expired sessions.
func (r *SessionRepo) DeleteExpired(ctx context.Context) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()
	now := time.Now()
	for k, v := range r.db.sessions {
		if now.After(v.ExpiresAt) {
			delete(r.db.sessions, k)
		}
	}
	return nil
}
