package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"vitals/internal/domain"

	"golang.org/x/crypto/bcrypt"
)

type mockUserRepo struct {
	getByUsernameFn func(ctx context.Context, username string) (*domain.User, error)
	getByIDFn       func(ctx context.Context, id int64) (*domain.User, error)
	createFn        func(ctx context.Context, username, passwordHash string) (*domain.User, error)
	countFn         func(ctx context.Context) (int, error)
}

func (m *mockUserRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	if m.getByUsernameFn != nil {
		return m.getByUsernameFn(ctx, username)
	}
	return nil, errors.New("not found")
}

func (m *mockUserRepo) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, errors.New("not found")
}

func (m *mockUserRepo) Create(ctx context.Context, username, passwordHash string) (*domain.User, error) {
	if m.createFn != nil {
		return m.createFn(ctx, username, passwordHash)
	}
	return &domain.User{ID: 1, Username: username, PasswordHash: passwordHash}, nil
}

func (m *mockUserRepo) Count(ctx context.Context) (int, error) {
	if m.countFn != nil {
		return m.countFn(ctx)
	}
	return 0, nil
}

type mockSessionRepo struct {
	createFn        func(ctx context.Context, userID int64, token, userAgent, ip string, expiresAt time.Time) error
	getByTokenFn    func(ctx context.Context, token string) (*domain.Session, error)
	deleteFn        func(ctx context.Context, token string) error
	deleteExpiredFn func(ctx context.Context) error
}

func (m *mockSessionRepo) Create(ctx context.Context, userID int64, token, userAgent, ip string, expiresAt time.Time) error {
	if m.createFn != nil {
		return m.createFn(ctx, userID, token, userAgent, ip, expiresAt)
	}
	return nil
}

func (m *mockSessionRepo) GetByToken(ctx context.Context, token string) (*domain.Session, error) {
	if m.getByTokenFn != nil {
		return m.getByTokenFn(ctx, token)
	}
	return nil, errors.New("not found")
}

func (m *mockSessionRepo) Delete(ctx context.Context, token string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, token)
	}
	return nil
}

func (m *mockSessionRepo) DeleteExpired(ctx context.Context) error {
	if m.deleteExpiredFn != nil {
		return m.deleteExpiredFn(ctx)
	}
	return nil
}

func TestAuthService_Login_Success(t *testing.T) {
	ctx := context.Background()
	password := "testpass123"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	users := &mockUserRepo{
		getByUsernameFn: func(ctx context.Context, username string) (*domain.User, error) {
			return &domain.User{
				ID:           1,
				Username:     "testuser",
				PasswordHash: string(hash),
			}, nil
		},
	}

	sessions := &mockSessionRepo{
		createFn: func(ctx context.Context, userID int64, token, userAgent, ip string, expiresAt time.Time) error {
			if userID != 1 {
				t.Errorf("expected userID 1, got %d", userID)
			}
			if token == "" {
				t.Error("token should not be empty")
			}
			return nil
		},
	}

	svc := NewAuthService(users, sessions)
	// Add userAgent="test-agent", ip="127.0.0.1"
	token, err := svc.Login(ctx, "testuser", password, "test-agent", "127.0.0.1")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token == "" {
		t.Error("expected token, got empty string")
	}
}

func TestAuthService_Login_InvalidPassword(t *testing.T) {
	ctx := context.Background()
	hash, _ := bcrypt.GenerateFromPassword([]byte("correctpass"), bcrypt.DefaultCost)

	users := &mockUserRepo{
		getByUsernameFn: func(ctx context.Context, username string) (*domain.User, error) {
			return &domain.User{
				ID:           1,
				Username:     "testuser",
				PasswordHash: string(hash),
			}, nil
		},
	}

	sessions := &mockSessionRepo{}
	svc := NewAuthService(users, sessions)

	// Add userAgent="test-agent", ip="127.0.0.1"
	_, err := svc.Login(ctx, "testuser", "wrongpass", "test-agent", "127.0.0.1")
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthService_ValidateSession_Valid(t *testing.T) {
	ctx := context.Background()
	token := "validtoken"
	userAgent := "test-agent"

	sessions := &mockSessionRepo{
		getByTokenFn: func(ctx context.Context, tok string) (*domain.Session, error) {
			return &domain.Session{
				Token:     token,
				UserID:    1,
				UserAgent: userAgent,
				ExpiresAt: time.Now().Add(1 * time.Hour),
			}, nil
		},
	}

	users := &mockUserRepo{
		getByIDFn: func(ctx context.Context, id int64) (*domain.User, error) {
			return &domain.User{
				ID:       1,
				Username: "testuser",
			}, nil
		},
	}

	svc := NewAuthService(users, sessions)
	// Add userAgent="test-agent"
	user, err := svc.ValidateSession(ctx, token, userAgent)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user.Username != "testuser" {
		t.Errorf("expected username 'testuser', got %s", user.Username)
	}
}

func TestAuthService_ValidateSession_Expired(t *testing.T) {
	ctx := context.Background()
	token := "expiredtoken"
	userAgent := "test-agent"

	deleted := false
	sessions := &mockSessionRepo{
		getByTokenFn: func(ctx context.Context, tok string) (*domain.Session, error) {
			return &domain.Session{
				Token:     token,
				UserID:    1,
				UserAgent: userAgent,
				ExpiresAt: time.Now().Add(-1 * time.Hour),
			}, nil
		},
		deleteFn: func(ctx context.Context, tok string) error {
			deleted = true
			return nil
		},
	}

	users := &mockUserRepo{}
	svc := NewAuthService(users, sessions)

	// Add userAgent="test-agent"
	_, err := svc.ValidateSession(ctx, token, userAgent)
	if err != ErrSessionExpired {
		t.Errorf("expected ErrSessionExpired, got %v", err)
	}
	if !deleted {
		t.Error("expected session to be deleted")
	}
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	ctx := context.Background()

	users := &mockUserRepo{
		getByUsernameFn: func(ctx context.Context, username string) (*domain.User, error) {
			return nil, nil
		},
	}

	sessions := &mockSessionRepo{}

	svc := NewAuthService(users, sessions)

	_, err := svc.Login(ctx, "nonexistent", "password", "agent", "127.0.0.1")
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}
