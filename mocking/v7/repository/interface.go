package repository

import (
	"context"
	"demo/entity"
)

type IUserRepository interface {
	// db tranction
	Transaction(context.Context, func(context.Context) error) error
	GetUser(context.Context, *entity.User) error
	UpdateUsers(context.Context, []entity.User) error
}
