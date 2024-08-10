package apn

import "context"

type Notification struct {
}

func (s *Service) SendNotification(ctx context.Context, n Notification) error {
	return nil
}
