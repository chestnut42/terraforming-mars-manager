package httpx

import (
	"fmt"
	"io"
	"net/http"
)

func CheckResponse(r *http.Response) error {
	if r.StatusCode != http.StatusOK {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}
		return fmt.Errorf("failed to send notification: status %d, body: %s", r.StatusCode, body)
	}
	return nil
}
