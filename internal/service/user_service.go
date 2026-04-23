package service

import (
	"errors"
	"fmt"

	"crypto-tracker-trader/internal/model"

	"golang.org/x/crypto/bcrypt"
)

// UserService implements the UserManager interface.
type UserService struct {
	userStore UserStoreInterface
}

// NewUserService creates a new UserService.
func NewUserService(us UserStoreInterface) *UserService {
	return &UserService{
		userStore: us,
	}
}

// RegisterUser registers a new user.
func (s *UserService) RegisterUser(username, email, password string) (*model.User, error) {
	// Check if user already exists
	existingUser, _, err := s.userStore.GetUserByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, errors.New("username already taken")
	}

	existingUser, _, err = s.userStore.GetUserByEmail(email)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, errors.New("email already taken")
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &model.User{
		Username: username,
		Email:    email,
		IsActive: true,
	}

	credential := &model.UserCredential{
		PasswordHash: string(hashedPassword),
		MFAEnabled:   false,
	}

	authProvider := &model.UserAuthProvider{
		Provider:       "local",
		ProviderUserID: username, // For local, ProviderUserID can be the username or a generated ID
	}

	// Store the user, credential, and auth provider
	if err := s.userStore.CreateUser(user, credential, authProvider); err != nil {
		return nil, fmt.Errorf("failed to create user in store: %w", err)
	}

	return user, nil
}

// LoginUser authenticates a user.
func (s *UserService) LoginUser(email, password string) (*model.User, error) {
	user, credential, err := s.userStore.GetUserByEmail(email)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}
	if user == nil {
		return nil, errors.New("invalid credentials") // User not found
	}

	// Compare the provided password with the stored hashed password
	if err := bcrypt.CompareHashAndPassword([]byte(credential.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials") // Password mismatch
	}

	return user, nil
}

// GetUserByID retrieves a user by their ID.
func (s *UserService) GetUserByID(userID uint64) (*model.User, error) {
	user, _, err := s.userStore.GetUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user by ID: %w", err)
	}
	if user == nil {
		return nil, errors.New("user not found")
	}
	return user, nil
}
