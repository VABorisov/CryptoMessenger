package credentialsstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/VABorisov/CryptoMessenger/internal/models"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

const redisKeyPrefix = "user:"

type RedisCredentialsStore struct {
	logger      *slog.Logger
	redisClient *redis.Client
}

func NewRedisCredentialsStore(ctx context.Context, logger *slog.Logger, redisAddr, redisPass string) (CredentialsStore, error) {
	logger.Info("connecting to Redis credentials store")
	redisOpt := &redis.Options{
		Addr:     redisAddr,
		Password: redisPass,
	}
	redisClient := redis.NewClient(redisOpt)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Error("failed to connect to Redis credentials store", "error", err)
		return nil, err
	}
	logger.Info("successfully connected to Redis credentials store")

	return &RedisCredentialsStore{
		logger:      logger,
		redisClient: redisClient,
	}, nil
}

func (s *RedisCredentialsStore) CreateUser(ctx context.Context, username, password string) error {
	s.logger.Info("creating new user in Redis credentials store", "username", username)
	exists, err := s.redisClient.Exists(ctx, redisKeyPrefix+username).Result()
	if err != nil {
		return err
	}

	if exists > 0 {
		s.logger.Error("user already exists in Redis credentials store", "username", username, "error", err)
		err := errors.New("user already exists")
		return fmt.Errorf("user creation error: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("failed to generate hash from password", "username", username, "error", err)
		return fmt.Errorf("user creation error: %w", err)
	}

	userModel := models.User{
		Username: username,
		Password: string(hash),
	}

	value, err := json.Marshal(userModel)
	if err != nil {
		s.logger.Error("failed to marshal user model", "username", username, "error", err)
		return fmt.Errorf("user creation error: %w", err)
	}

	err = s.redisClient.Set(ctx, redisKeyPrefix+username, value, 0).Err()
	if err != nil {
		s.logger.Error("failed to insert user data in Redis credentials store", "username", username, "error", err)
		return fmt.Errorf("user creation error: %w", err)
	}

	s.logger.Info("user successfully created in Redis credentials store", "username", username)

	return nil
}

func (s *RedisCredentialsStore) Authenticate(ctx context.Context, username, password string) (bool, error) {
	s.logger.Info("authenticating user in Redis credentials store", "username", username)
	user, err := s.GetUser(ctx, username)
	if err != nil {
		s.logger.Error("failed to obtain user data from Redis credentials store", "username", username, "error", err)
		return false, fmt.Errorf("user authenticating error: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		s.logger.Warn("failed to authentice user in Redis credentials store", "username", username, "error", "invalid passowrd")
		return false, nil
	}

	s.logger.Info("user succesfully authenticated in Redis credentials store", "username", username)
	return true, nil
}

func (s *RedisCredentialsStore) GetUser(ctx context.Context, username string) (*models.User, error) {
	s.logger.Info("obtaining user data from Redis credentials store", "username", username)
	value, err := s.redisClient.Get(ctx, redisKeyPrefix+username).Result()
	if err == redis.Nil {
		err := errors.New("user not found")
		s.logger.Info("failed to obtain user data from Redis credentials store", "username", username, "error", err)
		return nil, fmt.Errorf("user obtaining error: %w", err)
	} else if err != nil {
		s.logger.Info("failed to obtain user data from Redis credentials store", "username", username, "error", err)
		return nil, fmt.Errorf("user obtaining error: %w", err)
	}

	var userModel models.User
	if err := json.Unmarshal([]byte(value), &userModel); err != nil {
		s.logger.Info("failed to unmrashal user data", "username", username, "error", err)
		return nil, fmt.Errorf("user obtaining error: %w", err)
	}

	return &userModel, nil
}
