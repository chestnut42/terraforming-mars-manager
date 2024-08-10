package notifications

import (
	"context"
	"log/slog"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/chestnut42/terraforming-mars-manager/internal/framework/logx"
)

type Config struct {
	ActivityBuffer time.Duration
	ScanInterval   time.Duration
	WorkersCount   int
}

type Storage interface {
	GetActiveUsers(ctx context.Context, activityBuffer time.Duration) ([]string, error)
}

type Service struct {
	cfg     Config
	storage Storage

	users chan string
}

func NewService(cfg Config, storage Storage) *Service {
	return &Service{
		cfg:     cfg,
		storage: storage,

		users: make(chan string),
	}
}

func (s *Service) Run(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return s.scanUsers(ctx)
	})

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
