package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"messenger/internal/models"
	"messenger/internal/repositories"

	webpush "github.com/SherClockHolmes/webpush-go"
	"go.uber.org/zap"
)

// webpushSendTimeout caps a single WebPush delivery so a slow/hung push endpoint
// doesn't leak goroutines while we keep firing per-subscription sends.
const webpushSendTimeout = 10 * time.Second

// PushPayload represents the JSON payload sent to the browser
type PushPayload struct {
	Title  string `json:"title"`
	Body   string `json:"body"`
	ChatID uint   `json:"chat_id"`
	UserID uint   `json:"user_id"`
	Tag    string `json:"tag,omitempty"`
}

// PushNotificationService fans push notifications out to Web Push (VAPID) and FCM (mobile).
type PushNotificationService struct {
	logger          *zap.Logger
	pushSubRepo     *repositories.PushSubscriptionRepo
	fcm             *FCMService
	httpClient      *http.Client
	vapidPublicKey  string
	vapidPrivateKey string
	vapidSubject    string
	enabled         bool
}

// NewPushNotificationService creates a new push notification service.
// If VAPID keys are not configured, push notifications are disabled (graceful degradation).
func NewPushNotificationService(
	logger *zap.Logger,
	pushSubRepo *repositories.PushSubscriptionRepo,
	vapidPublicKey, vapidPrivateKey, vapidSubject string,
) *PushNotificationService {
	enabled := vapidPublicKey != "" && vapidPrivateKey != "" && vapidSubject != ""
	if !enabled {
		logger.Warn("push notifications disabled: VAPID keys not configured")
	}
	return &PushNotificationService{
		logger:          logger,
		pushSubRepo:     pushSubRepo,
		httpClient:      &http.Client{Timeout: webpushSendTimeout},
		vapidPublicKey:  vapidPublicKey,
		vapidPrivateKey: vapidPrivateKey,
		vapidSubject:    vapidSubject,
		enabled:         enabled,
	}
}

// SetFCMService attaches an FCMService for mobile push fan-out. Optional.
func (s *PushNotificationService) SetFCMService(fcm *FCMService) {
	s.fcm = fcm
}

// IsEnabled returns whether at least one push channel (VAPID or FCM) is configured.
func (s *PushNotificationService) IsEnabled() bool {
	return s.enabled || (s.fcm != nil && s.fcm.IsEnabled())
}

// GetVAPIDPublicKey returns the public key for frontend subscription
func (s *PushNotificationService) GetVAPIDPublicKey() string {
	return s.vapidPublicKey
}

// maxSubscriptionsPerUser is the maximum number of push subscriptions allowed per user.
const maxSubscriptionsPerUser = 10

// Subscribe stores a push subscription for a user.
// Enforces a per-user limit to prevent abuse.
func (s *PushNotificationService) Subscribe(ctx context.Context, userID uint, endpoint, p256dh, auth string) error {
	// Check subscription limit before creating a new one
	count, err := s.pushSubRepo.CountByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to check subscription count: %w", err)
	}
	if count >= maxSubscriptionsPerUser {
		return fmt.Errorf("subscription limit reached (max %d)", maxSubscriptionsPerUser)
	}

	sub := &models.PushSubscription{
		UserID:   userID,
		Endpoint: endpoint,
		P256dh:   p256dh,
		Auth:     auth,
	}
	return s.pushSubRepo.Upsert(ctx, sub)
}

// Unsubscribe removes a push subscription
func (s *PushNotificationService) Unsubscribe(ctx context.Context, userID uint, endpoint string) error {
	return s.pushSubRepo.DeleteByEndpoint(ctx, userID, endpoint)
}

// HasSubscriptions checks if a user has any push subscriptions
func (s *PushNotificationService) HasSubscriptions(ctx context.Context, userID uint) (bool, error) {
	return s.pushSubRepo.ExistsByUser(ctx, userID)
}

// SendNotification fans a notification out to all Web Push subscriptions and FCM tokens of a user.
func (s *PushNotificationService) SendNotification(ctx context.Context, recipientUserID uint, payload PushPayload) error {
	// FCM fan-out (mobile) runs in parallel to VAPID (web).
	if s.fcm != nil && s.fcm.IsEnabled() {
		if err := s.fcm.SendNotification(ctx, recipientUserID, payload); err != nil {
			s.logger.Warn("FCM fan-out failed", zap.Error(err), zap.Uint("user_id", recipientUserID))
		}
	}

	if !s.enabled {
		return nil
	}

	subs, err := s.pushSubRepo.GetByUser(ctx, recipientUserID)
	if err != nil {
		return fmt.Errorf("failed to get push subscriptions: %w", err)
	}

	if len(subs) == 0 {
		return nil
	}

	s.logger.Info("sending push notification",
		zap.Uint("recipient_id", recipientUserID),
		zap.Int("subscription_count", len(subs)),
	)

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal push payload: %w", err)
	}

	// Use background context so goroutines don't depend on caller's lifecycle.
	bgCtx := context.Background()
	for _, sub := range subs {
		go s.sendToSubscription(bgCtx, sub, payloadJSON) //nolint:contextcheck // intentionally detached from caller context
	}

	return nil
}

// sendToSubscription sends push to a single subscription endpoint
func (s *PushNotificationService) sendToSubscription(ctx context.Context, sub models.PushSubscription, payload []byte) {
	subscription := &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys: webpush.Keys{
			P256dh: sub.P256dh,
			Auth:   sub.Auth,
		},
	}

	resp, err := webpush.SendNotification(payload, subscription, &webpush.Options{
		HTTPClient:      s.httpClient,
		Subscriber:      s.vapidSubject,
		VAPIDPublicKey:  s.vapidPublicKey,
		VAPIDPrivateKey: s.vapidPrivateKey,
		TTL:             86400, // 24 hours
		Urgency:         webpush.UrgencyHigh,
	})
	if err != nil {
		s.logger.Error("failed to send push notification",
			zap.Error(err),
			zap.String("endpoint", sub.Endpoint),
			zap.Uint("user_id", sub.UserID),
		)
		return
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusCreated, http.StatusOK:
		s.logger.Debug("push notification delivered",
			zap.Uint("user_id", sub.UserID),
		)
	case http.StatusGone:
		s.logger.Info("push subscription expired, removing",
			zap.String("endpoint", sub.Endpoint),
			zap.Uint("user_id", sub.UserID),
		)
		if err := s.pushSubRepo.DeleteByEndpointGlobal(ctx, sub.Endpoint); err != nil {
			s.logger.Error("failed to remove expired subscription", zap.Error(err))
		}
	case http.StatusNotFound:
		s.logger.Info("push subscription invalid, removing", zap.String("endpoint", sub.Endpoint))
		if err := s.pushSubRepo.DeleteByEndpointGlobal(ctx, sub.Endpoint); err != nil {
			s.logger.Error("failed to remove invalid subscription", zap.Error(err))
		}
	default:
		s.logger.Warn("unexpected push response",
			zap.Int("status", resp.StatusCode),
			zap.String("endpoint", sub.Endpoint),
		)
	}
}
