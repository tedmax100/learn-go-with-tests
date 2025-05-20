package service

import (
	"context"
	"demo/entity"
	"demo/repository"
)

type UserService struct {
	repo repository.IUserRepository
}

func New(repo repository.IUserRepository) *UserService {
	return &UserService{
		repo: repo,
	}
}

func (u *UserService) GetUser(ctx context.Context, user *entity.User) error {
	return u.repo.GetUser(ctx, user)
}

func (u *UserService) UpdateUsers(ctx context.Context, users []entity.User) error {
	return u.repo.Transaction(ctx, func(ctx context.Context) error {
		if err := u.repo.UpdateUsers(ctx, users); err != nil {
			return err
		}
		return nil
	})
}
