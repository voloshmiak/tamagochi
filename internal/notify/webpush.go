package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/SherClockHolmes/webpush-go"
)

const pushTTL = 12 * 60 * 60

type WebPushSender struct {
	PublicKey  string
	PrivateKey string
	Subject    string
	Client     webpush.HTTPClient
}

type pushPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
	Tag   string `json:"tag,omitempty"`
	Kind  Kind   `json:"kind"`
}

func (s WebPushSender) Send(ctx context.Context, sub Subscription, n Notification) error {
	msg, err := json.Marshal(pushPayload{Title: n.Title, Body: n.Body, URL: n.URL, Tag: n.Tag, Kind: n.Kind})
	if err != nil {
		return err
	}
	resp, err := webpush.SendNotificationWithContext(ctx, msg, &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys:     webpush.Keys{P256dh: sub.P256dh, Auth: sub.Auth},
	}, &webpush.Options{
		HTTPClient:      s.Client,
		Subscriber:      s.Subject,
		VAPIDPublicKey:  s.PublicKey,
		VAPIDPrivateKey: s.PrivateKey,
		TTL:             pushTTL,
		Urgency:         webpush.UrgencyNormal,
	})
	if err != nil {
		return fmt.Errorf("web push: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))
	switch {
	case resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound:
		return fmt.Errorf("%w: push service answered %d", ErrGone, resp.StatusCode)
	case resp.StatusCode >= 400:
		return fmt.Errorf("web push: push service answered %d", resp.StatusCode)
	}
	return nil
}

type LogSender struct{}

func (LogSender) Send(ctx context.Context, sub Subscription, n Notification) error {
	slog.InfoContext(ctx, "push notification (VAPID keys not configured)", "user", n.UserID, "kind", n.Kind, "title", n.Title)
	return nil
}
