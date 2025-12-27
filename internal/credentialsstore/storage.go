package credentialsstore

import (
	"context"

	"github.com/VABorisov/CryptoMessenger/internal/models"
)

type CredentialsStore interface {
	CreateUser(ctx context.Context, username, password string) error
	Authenticate(ctx context.Context, username, password string) (bool, error)
	GetUser(ctx context.Context, username string) (*models.User, error)
}
