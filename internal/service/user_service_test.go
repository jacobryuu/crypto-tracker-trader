package service

import (
	"errors"
	"testing"

	"crypto-tracker-trader/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

type MockUserStore struct {
	mock.Mock
}

func (m *MockUserStore) CreateUser(user *model.User, credential *model.UserCredential, authProvider *model.UserAuthProvider) error {
	args := m.Called(user, credential, authProvider)
	return args.Error(0)
}

func (m *MockUserStore) GetUserByUsername(username string) (*model.User, *model.UserCredential, error) {
	args := m.Called(username)
	user, _ := args.Get(0).(*model.User)
	credential, _ := args.Get(1).(*model.UserCredential)
	return user, credential, args.Error(2)
}

func (m *MockUserStore) GetUserByEmail(email string) (*model.User, *model.UserCredential, error) {
	args := m.Called(email)
	user, _ := args.Get(0).(*model.User)
	credential, _ := args.Get(1).(*model.UserCredential)
	return user, credential, args.Error(2)
}

func (m *MockUserStore) GetUserByID(userID uint64) (*model.User, *model.UserCredential, error) {
	args := m.Called(userID)
	user, _ := args.Get(0).(*model.User)
	credential, _ := args.Get(1).(*model.UserCredential)
	return user, credential, args.Error(2)
}

func TestUserService_RegisterUser_Success(t *testing.T) {
	mockUserStore := new(MockUserStore)
	userService := NewUserService(mockUserStore)

	username := "testuser"
	email := "test@example.com"
	password := "password123"

	// Mock GetUserByUsername and GetUserByEmail to return nil (user not found)
	mockUserStore.On("GetUserByUsername", username).Return((*model.User)(nil), (*model.UserCredential)(nil), nil).Once()
	mockUserStore.On("GetUserByEmail", email).Return((*model.User)(nil), (*model.UserCredential)(nil), nil).Once()

	// Mock CreateUser to return no error and set IDs
	mockUserStore.On("CreateUser", mock.AnythingOfType("*model.User"), mock.AnythingOfType("*model.UserCredential"), mock.AnythingOfType("*model.UserAuthProvider")).Return(nil).Run(func(args mock.Arguments) {
		userArg := args.Get(0).(*model.User)
		userArg.ID = 1 // Simulate ID being set by DB
		credentialArg := args.Get(1).(*model.UserCredential)
		credentialArg.ID = 1
		authProviderArg := args.Get(2).(*model.UserAuthProvider)
		authProviderArg.ID = 1
	}).Once()

	user, err := userService.RegisterUser(username, email, password)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, username, user.Username)
	assert.Equal(t, email, user.Email)
	assert.NotZero(t, user.ID)

	// Password hash is now in UserCredential, which is not directly returned by RegisterUser
	// So we can only verify the call to CreateUser included a valid hashed password
	mockUserStore.AssertCalled(t, "CreateUser",
		mock.AnythingOfType("*model.User"),
		mock.MatchedBy(func(cred *model.UserCredential) bool {
			err := bcrypt.CompareHashAndPassword([]byte(cred.PasswordHash), []byte(password))
			return err == nil
		}),
		mock.AnythingOfType("*model.UserAuthProvider"),
	)
	mockUserStore.AssertExpectations(t)
}

func TestUserService_RegisterUser_DuplicateUsername(t *testing.T) {
	mockUserStore := new(MockUserStore)
	userService := NewUserService(mockUserStore)

	username := "existinguser"
	email := "existing@example.com"
	password := "password123"

	// Mock GetUserByUsername to return an existing user
	mockUserStore.On("GetUserByUsername", username).Return(&model.User{ID: 1, Username: username, Email: email}, &model.UserCredential{}, nil).Once()

	user, err := userService.RegisterUser(username, email, password)
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Equal(t, "username already taken", err.Error())

	mockUserStore.AssertExpectations(t)
	mockUserStore.AssertNotCalled(t, "CreateUser") // Should not call CreateUser
}

func TestUserService_LoginUser_Success(t *testing.T) {
	mockUserStore := new(MockUserStore)
	userService := NewUserService(mockUserStore)

	email := "test@example.com"
	password := "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	// Mock GetUserByEmail to return a user with hashed password in credential
	mockUserStore.On("GetUserByEmail", email).Return(
		&model.User{ID: 1, Email: email, Username: "testuser"},
		&model.UserCredential{PasswordHash: string(hashedPassword)},
		nil,
	).Once()

	user, err := userService.LoginUser(email, password)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, email, user.Email)
	assert.Equal(t, uint64(1), user.ID)

	mockUserStore.AssertExpectations(t)
}

func TestUserService_LoginUser_InvalidCredentials(t *testing.T) {
	mockUserStore := new(MockUserStore)
	userService := NewUserService(mockUserStore)

	email := "test@example.com"
	password := "wrongpassword"
	correctPassword := "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(correctPassword), bcrypt.DefaultCost)

	// Mock GetUserByEmail to return a user with correct hashed password in credential
	mockUserStore.On("GetUserByEmail", email).Return(
		&model.User{ID: 1, Email: email, Username: "testuser"},
		&model.UserCredential{PasswordHash: string(hashedPassword)},
		nil,
	).Once()

	user, err := userService.LoginUser(email, password)
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Equal(t, "invalid credentials", err.Error())

	mockUserStore.AssertExpectations(t)
}

func TestUserService_LoginUser_UserNotFound(t *testing.T) {
	mockUserStore := new(MockUserStore)
	userService := NewUserService(mockUserStore)

	email := "nonexistent@example.com"
	password := "password123"

	// Mock GetUserByEmail to return nil for user and credential
	mockUserStore.On("GetUserByEmail", email).Return((*model.User)(nil), (*model.UserCredential)(nil), nil).Once()

	user, err := userService.LoginUser(email, password)
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Equal(t, "invalid credentials", err.Error())

	mockUserStore.AssertExpectations(t)
}

func TestUserService_GetUserByID_Success(t *testing.T) {
	mockUserStore := new(MockUserStore)
	userService := NewUserService(mockUserStore)

	userID := uint64(1)
	expectedUser := &model.User{ID: userID, Username: "testuser", Email: "test@example.com"}

	mockUserStore.On("GetUserByID", userID).Return(expectedUser, &model.UserCredential{}, nil).Once()

	user, err := userService.GetUserByID(userID)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, expectedUser.ID, user.ID)
	assert.Equal(t, expectedUser.Username, user.Username)
	assert.Equal(t, expectedUser.Email, user.Email)

	mockUserStore.AssertExpectations(t)
}

func TestUserService_GetUserByID_NotFound(t *testing.T) {
	mockUserStore := new(MockUserStore)
	userService := NewUserService(mockUserStore)

	userID := uint64(999) // Non-existent ID

	mockUserStore.On("GetUserByID", userID).Return((*model.User)(nil), (*model.UserCredential)(nil), nil).Once()

	user, err := userService.GetUserByID(userID)
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Equal(t, "user not found", err.Error())

	mockUserStore.AssertExpectations(t)
}

func TestUserService_GetUserByID_StoreError(t *testing.T) {
	mockUserStore := new(MockUserStore)
	userService := NewUserService(mockUserStore)

	userID := uint64(1)
	storeError := errors.New("database error")

	mockUserStore.On("GetUserByID", userID).Return((*model.User)(nil), (*model.UserCredential)(nil), storeError).Once()

	user, err := userService.GetUserByID(userID)
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "failed to retrieve user by ID")
	assert.Contains(t, err.Error(), storeError.Error())

	mockUserStore.AssertExpectations(t)
}
