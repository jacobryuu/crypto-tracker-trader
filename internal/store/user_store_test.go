package store

import (
	"context"
	"os"
	"testing"

	"crypto-tracker-trader/internal/model"

	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func getUserTestDatabaseURL() string {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		return "postgres://ctt:password@localhost:5532/crypto_test?sslmode=disable" // Use a separate test DB
	}
	return url
}

func setupUserStoreTestDB(t *testing.T) *pgx.Conn {
	dbURL := getUserTestDatabaseURL()
	conn, err := pgx.Connect(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("Unable to connect to test database: %v", err)
	}

	// Clear existing tables in correct order due to foreign keys
	_, err = conn.Exec(context.Background(), "DROP TABLE IF EXISTS user_defi_positions CASCADE;")
	assert.NoError(t, err)
	_, err = conn.Exec(context.Background(), "DROP TABLE IF EXISTS user_nfts CASCADE;")
	assert.NoError(t, err)
	_, err = conn.Exec(context.Background(), "DROP TABLE IF EXISTS user_assets CASCADE;")
	assert.NoError(t, err)
	_, err = conn.Exec(context.Background(), "DROP TABLE IF EXISTS user_wallets CASCADE;")
	assert.NoError(t, err)
	_, err = conn.Exec(context.Background(), "DROP TABLE IF EXISTS user_auth_providers CASCADE;")
	assert.NoError(t, err)
	_, err = conn.Exec(context.Background(), "DROP TABLE IF EXISTS user_credentials CASCADE;")
	assert.NoError(t, err)
	_, err = conn.Exec(context.Background(), "DROP TABLE IF EXISTS users CASCADE;")
	assert.NoError(t, err)
	_, err = conn.Exec(context.Background(), "DROP TABLE IF EXISTS portfolio_assets CASCADE;")
	assert.NoError(t, err)
	_, err = conn.Exec(context.Background(), "DROP TABLE IF EXISTS portfolio_snapshots CASCADE;")
	assert.NoError(t, err)

	// Create tables (users, user_credentials, user_auth_providers)
	_, err = conn.Exec(context.Background(), `
		CREATE TABLE users (
			id BIGSERIAL PRIMARY KEY,
			username VARCHAR(50) UNIQUE NOT NULL,
			email VARCHAR(255) UNIQUE NOT NULL,
			is_active BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		);

		CREATE TABLE user_credentials (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL,
			password_hash VARCHAR(255),
			mfa_enabled BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);

		CREATE TABLE user_auth_providers (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL,
			provider VARCHAR(30) NOT NULL,
			provider_user_id VARCHAR(255) NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);
    `)
	assert.NoError(t, err)

	return conn
}

func teardownUserStoreTestDB(t *testing.T, conn *pgx.Conn) {
	_, err := conn.Exec(context.Background(), "DROP TABLE IF EXISTS users CASCADE;")
	assert.NoError(t, err)
	if err := conn.Close(context.Background()); err != nil {
		t.Fatalf("Error closing database connection: %v", err)
	}
}

func TestUserStore_CreateUser(t *testing.T) {
	conn := setupUserStoreTestDB(t)
	defer teardownUserStoreTestDB(t, conn)

	store := &UserStore{db: conn}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := &model.User{
		Username: "testuser",
		Email:    "test@example.com",
		IsActive: true,
	}
	credential := &model.UserCredential{
		PasswordHash: string(hashedPassword),
		MFAEnabled:   false,
	}
	authProvider := &model.UserAuthProvider{
		Provider:       "local",
		ProviderUserID: "testuser_id",
	}

	err := store.CreateUser(user, credential, authProvider)
	assert.NoError(t, err)
	assert.NotZero(t, user.ID)         // User ID should be set after creation
	assert.NotZero(t, credential.ID)   // Credential ID should be set
	assert.NotZero(t, authProvider.ID) // AuthProvider ID should be set

	// Verify user, credential, and auth provider exist in DB
	retrievedUser, retrievedCredential, err := store.GetUserByID(user.ID)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedUser)
	assert.NotNil(t, retrievedCredential)
	assert.Equal(t, user.Username, retrievedUser.Username)
	assert.Equal(t, user.Email, retrievedUser.Email)
	assert.Equal(t, credential.PasswordHash, retrievedCredential.PasswordHash)
	assert.Equal(t, user.ID, retrievedCredential.UserID)

	retrievedUserByUsername, retrievedCredentialByUsername, err := store.GetUserByUsername("testuser")
	assert.NoError(t, err)
	assert.NotNil(t, retrievedUserByUsername)
	assert.NotNil(t, retrievedCredentialByUsername)
	assert.Equal(t, user.Username, retrievedUserByUsername.Username)
	assert.Equal(t, user.Email, retrievedUserByUsername.Email)
	assert.Equal(t, credential.PasswordHash, retrievedCredentialByUsername.PasswordHash)
	assert.Equal(t, user.ID, retrievedCredentialByUsername.UserID)

	// Verify AuthProvider (requires a separate query as it's not joined in GetUserByID/Username)
	var count int
	err = conn.QueryRow(context.Background(), "SELECT COUNT(*) FROM user_auth_providers WHERE user_id = $1 AND provider = $2 AND provider_user_id = $3",
		user.ID, authProvider.Provider, authProvider.ProviderUserID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestUserStore_GetUserByUsername_NotFound(t *testing.T) {
	conn := setupUserStoreTestDB(t)
	defer teardownUserStoreTestDB(t, conn)

	store := &UserStore{db: conn}

	user, credential, err := store.GetUserByUsername("nonexistent")
	assert.NoError(t, err) // pgx.ErrNoRows is handled to return nil, nil
	assert.Nil(t, user)
	assert.Nil(t, credential)
}

func TestUserStore_GetUserByID_NotFound(t *testing.T) {
	conn := setupUserStoreTestDB(t)
	defer teardownUserStoreTestDB(t, conn)

	store := &UserStore{db: conn}

	user, credential, err := store.GetUserByID(999) // Non-existent ID
	assert.NoError(t, err)
	assert.Nil(t, user)
	assert.Nil(t, credential)
}

func TestUserStore_CreateUser_DuplicateUsername(t *testing.T) {
	conn := setupUserStoreTestDB(t)
	defer teardownUserStoreTestDB(t, conn)

	store := &UserStore{db: conn}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user1 := &model.User{
		Username: "duplicateuser",
		Email:    "duplicate@example.com",
		IsActive: true,
	}
	credential1 := &model.UserCredential{
		PasswordHash: string(hashedPassword),
		MFAEnabled:   false,
	}
	authProvider1 := &model.UserAuthProvider{
		Provider:       "local",
		ProviderUserID: "duplicateuser_id",
	}
	err := store.CreateUser(user1, credential1, authProvider1)
	assert.NoError(t, err)

	user2 := &model.User{
		Username: "duplicateuser", // Duplicate username
		Email:    "another@example.com",
		IsActive: true,
	}
	credential2 := &model.UserCredential{
		PasswordHash: string(hashedPassword),
		MFAEnabled:   false,
	}
	authProvider2 := &model.UserAuthProvider{
		Provider:       "local",
		ProviderUserID: "anotheruser_id",
	}
	err = store.CreateUser(user2, credential2, authProvider2)
	assert.Error(t, err) // Expect an error for duplicate username
	assert.Contains(t, err.Error(), "duplicate key value violates unique constraint \"users_username_key\"")
}
