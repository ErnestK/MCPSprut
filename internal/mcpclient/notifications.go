package mcpclient

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func (c *Client) SubscribeNotifications(ctx context.Context, serverURL string) (<-chan string, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Accept", "text/event-stream")

	sseClient := &http.Client{}
	resp, err := sseClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("connect sse: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("sse status: %d", resp.StatusCode)
	}

	ch := make(chan string, 16)

	go func() {
		defer resp.Body.Close()
		defer close(ch)

		scanner := bufio.NewScanner(resp.Body)
		var dataLine string

		for scanner.Scan() {
			line := scanner.Text()

			if strings.HasPrefix(line, "data: ") {
				dataLine = strings.TrimPrefix(line, "data: ")
			} else if line == "" && dataLine != "" {
				var notification struct {
					Method string `json:"method"`
				}
				if json.Unmarshal([]byte(dataLine), &notification) == nil && notification.Method != "" {
					select {
					case ch <- notification.Method:
					case <-ctx.Done():
						return
					}
				}
				dataLine = ""
			}
		}
	}()

	return ch, nil
}
