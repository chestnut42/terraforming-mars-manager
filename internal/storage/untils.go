package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/chestnut42/terraforming-mars-manager/internal/framework/logx"
)

func (s *Storage) withTX(ctx context.Context, f func(ctx context.Context, tx *sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	if err := f(ctx, tx); err != nil {
		if err := tx.Rollback(); err != nil {
			logx.Logger(ctx).Info("failed to rollback transaction", slog.Any("error", err))
		}
		return err
	}
	return tx.Commit()
}
