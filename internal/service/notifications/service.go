package notifications

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/chestnut42/terraforming-mars-manager/internal/client/apn"
	"github.com/chestnut42/terraforming-mars-manager/internal/framework/logx"
	"github.com/chestnut42/terraforming-mars-manager/internal/service/game"
	"github.com/chestnut42/terraforming-mars-manager/internal/storage"
)

type Config struct {
	ActivityBuffer time.Duration
	ScanInterval   time.Duration
	WorkersCount   int
}

type Storage interface {
	GetActiveUsers(ctx context.Context, activityBuffer time.Duration) ([]string, error)
	GetUserById(ctx context.Context, userId string) (*storage.User, error)
	UpdateSentNotification(ctx context.Context, userId string, updater storage.SentNotificationUpdater) error
}

type GameService interface {
	GetUserGames(ctx context.Context, userId string) ([]*game.UserGame, error)
}

type Notifier interface {
	SendNotification(ctx context.Context, device []byte, n apn.Notification) error
}

type Service struct {
	cfg      Config
	storage  Storage
	game     GameService
	notifier Notifier

	users chan string
}

func NewService(cfg Config, storage Storage, game GameService, notifier Notifier) *Service {
	return &Service{
		cfg:      cfg,
		storage:  storage,
		game:     game,
		notifier: notifier,

		users: make(chan string),
	}
}

func (s *Service) NotifyUser(ctx context.Context, userId string) error {
	select {
	case s.users <- userId:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Service) Run(ctx context.Context) error {
	if s.cfg.WorkersCount <= 0 {
		return fmt.Errorf("workersCount must be greater than zero: %d", s.cfg.WorkersCount)
	}

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return s.scanUsers(ctx)
	})
	for i := 0; i < s.cfg.WorkersCount; i++ {
		eg.Go(func() error {
			return s.worker(ctx)
		})
	}
	return eg.Wait()
}

func (s *Service) scanUsers(ctx context.Context) error {
	for {
		users, err := s.storage.GetActiveUsers(ctx, s.cfg.ActivityBuffer)
		if err != nil {
			logx.Logger(ctx).Error("failed to get active users", slog.Any("error", err))
		}

		for _, u := range users {
			s.users <- u
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(s.cfg.ScanInterval):
		}
	}
}

func (s *Service) worker(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case uid := <-s.users:
			err := s.processUser(ctx, uid)
			if err != nil {
				logx.Logger(ctx).Error("failed to process user", slog.String("uid", uid), slog.Any("error", err))
			}
		}
	}
}
