package store

import (
	"context"
	"log"
	"time"

	"crypto-tracker-trader/internal/model"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// UserStore implements UserStoreInterface for PostgreSQL.
type UserStore struct {
	db *pgxpool.Pool
}

// NewUserStore creates a new UserStore instance.
func NewUserStore(db *pgxpool.Pool) *UserStore {
	return &UserStore{db: db}
}

// Close closes the database pool.
func (s *UserStore) Close() {
	s.db.Close()
	log.Printf("UserStore database pool closed")
}

// CreateUser inserts a new user, their credential, and auth provider into the database.
func (s *UserStore) CreateUser(user *model.User, credential *model.UserCredential, authProvider *model.UserAuthProvider) error {
	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if rErr := tx.Rollback(ctx); rErr != nil && rErr != pgx.ErrTxClosed {
			log.Printf("Rollback failed: %v", rErr)
		}
	}()

	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	err = tx.QueryRow(ctx,
		"INSERT INTO users (username, email, is_active, created_at, updated_at) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		user.Username, user.Email, user.IsActive, user.CreatedAt, user.UpdatedAt).Scan(&user.ID)
	if err != nil {
		return err
	}

	credential.UserID = user.ID
	credential.CreatedAt = time.Now()
	credential.UpdatedAt = time.Now()
	err = tx.QueryRow(ctx,
		"INSERT INTO user_credentials (user_id, password_hash, mfa_enabled, created_at, updated_at) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		credential.UserID, credential.PasswordHash, credential.MFAEnabled, credential.CreatedAt, credential.UpdatedAt).Scan(&credential.ID)
	if err != nil {
		return err
	}

	authProvider.UserID = user.ID
	authProvider.CreatedAt = time.Now()
	err = tx.QueryRow(ctx,
		"INSERT INTO user_auth_providers (user_id, provider, provider_user_id, created_at) VALUES ($1, $2, $3, $4) RETURNING id",
		authProvider.UserID, authProvider.Provider, authProvider.ProviderUserID, authProvider.CreatedAt).Scan(&authProvider.ID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// GetUserByUsername retrieves a user and their credential by username.
func (s *UserStore) GetUserByUsername(username string) (*model.User, *model.UserCredential, error) {
	ctx := context.Background()
	user := &model.User{}
	credential := &model.UserCredential{}
	err := s.db.QueryRow(ctx,
		`SELECT u.id, u.username, u.email, u.is_active, u.created_at, u.updated_at,
		c.id, c.password_hash, c.mfa_enabled, c.created_at, c.updated_at
		FROM users u JOIN user_credentials c ON u.id = c.user_id WHERE u.username = $1`,
		username).Scan(
		&user.ID, &user.Username, &user.Email, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
		&credential.ID, &credential.PasswordHash, &credential.MFAEnabled, &credential.CreatedAt, &credential.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil, nil // User not found
		}
		return nil, nil, err
	}
	return user, credential, nil
}

// GetUserByEmail retrieves a user and their credential by email.
func (s *UserStore) GetUserByEmail(email string) (*model.User, *model.UserCredential, error) {
	ctx := context.Background()
	user := &model.User{}
	credential := &model.UserCredential{}
	err := s.db.QueryRow(ctx,
		`SELECT u.id, u.username, u.email, u.is_active, u.created_at, u.updated_at,
		c.id, c.password_hash, c.mfa_enabled, c.created_at, c.updated_at
		FROM users u JOIN user_credentials c ON u.id = c.user_id WHERE u.email = $1`,
		email).Scan(
		&user.ID, &user.Username, &user.Email, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
		&credential.ID, &credential.PasswordHash, &credential.MFAEnabled, &credential.CreatedAt, &credential.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil, nil // User not found
		}
		return nil, nil, err
	}
	return user, credential, nil
}

// GetUserByID retrieves a user and their credential by ID.
func (s *UserStore) GetUserByID(id uint64) (*model.User, *model.UserCredential, error) {
	ctx := context.Background()
	user := &model.User{}
	credential := &model.UserCredential{}
	err := s.db.QueryRow(ctx,
		`SELECT u.id, u.username, u.email, u.is_active, u.created_at, u.updated_at,
		c.id, c.password_hash, c.mfa_enabled, c.created_at, c.updated_at
		FROM users u JOIN user_credentials c ON u.id = c.user_id WHERE u.id = $1`,
		id).Scan(
		&user.ID, &user.Username, &user.Email, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
		&credential.ID, &credential.PasswordHash, &credential.MFAEnabled, &credential.CreatedAt, &credential.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil, nil // User not found
		}
		return nil, nil, err
	}
	return user, credential, nil
}
