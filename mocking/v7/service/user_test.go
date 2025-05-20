package service_test

import (
	"context"
	"demo/entity"
	"demo/repository"
	"demo/service"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestGetUser(t *testing.T) {
	t.Run("should get user successfully", func(t *testing.T) {
		// Arrange
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repository.NewMockIUserRepository(ctrl)
		userId := uuid.New()
		expectedUser := &entity.User{
			Id:   userId,
			Name: "John Doe",
		}

		// mocking repository.GetUser
		mockRepo.EXPECT().
			GetUser(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, user *entity.User) error {
				user.Id = expectedUser.Id
				user.Name = expectedUser.Name
				return nil
			})

		userService := service.New(mockRepo)
		ctx := context.Background()
		user := &entity.User{Id: userId}

		// Act
		err := userService.GetUser(ctx, user)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedUser.Id, user.Id)
		assert.Equal(t, expectedUser.Name, user.Name)
	})

	t.Run("should return error when repository fails", func(t *testing.T) {
		// Arrange
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repository.NewMockIUserRepository(ctrl)
		expectedErr := errors.New("database error")

		// mock occur exception
		mockRepo.EXPECT().
			GetUser(gomock.Any(), gomock.Any()).
			Return(expectedErr)

		userService := service.New(mockRepo)
		ctx := context.Background()
		user := &entity.User{Id: uuid.New()}

		// Act
		err := userService.GetUser(ctx, user)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})
}

func TestUpdateUsers(t *testing.T) {
	t.Run("should update users successfully", func(t *testing.T) {
		// Arrange
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repository.NewMockIUserRepository(ctrl)
		users := []entity.User{
			{Id: uuid.New(), Name: "User 1"},
			{Id: uuid.New(), Name: "User 2"},
		}

		// mock transaction
		mockRepo.EXPECT().
			Transaction(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, txFn func(context.Context) error) error {
				return txFn(context.Background())
			})

		// invoke UpdateUsers in trancaction scope
		mockRepo.EXPECT().
			UpdateUsers(gomock.Any(), users).
			Return(nil)

		userService := service.New(mockRepo)
		ctx := context.Background()

		// Act
		err := userService.UpdateUsers(ctx, users)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("should return error when transaction fails", func(t *testing.T) {
		// Arrange
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repository.NewMockIUserRepository(ctrl)
		users := []entity.User{
			{Id: uuid.New(), Name: "User 1"},
		}
		expectedErr := errors.New("transaction error")

		// mock transaction fail
		mockRepo.EXPECT().
			Transaction(gomock.Any(), gomock.Any()).
			Return(expectedErr)

		userService := service.New(mockRepo)
		ctx := context.Background()

		// Act
		err := userService.UpdateUsers(ctx, users)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("should return error when update users fails", func(t *testing.T) {
		// Arrange
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repository.NewMockIUserRepository(ctrl)
		users := []entity.User{
			{Id: uuid.New(), Name: "User 1"},
		}
		expectedErr := errors.New("update error")

		// set paramater for Transaction
		mockRepo.EXPECT().
			Transaction(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, txFn func(context.Context) error) error {
				return txFn(context.Background())
			})

		// mocking UpdateUsers fail
		mockRepo.EXPECT().
			UpdateUsers(gomock.Any(), users).
			Return(expectedErr)

		userService := service.New(mockRepo)
		ctx := context.Background()

		// Act
		err := userService.UpdateUsers(ctx, users)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})
}
