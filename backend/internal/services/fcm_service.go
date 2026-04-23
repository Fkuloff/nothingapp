package services

import (
	"context"
	"fmt"
	"strconv"

	"messenger/internal/models"
	"messenger/internal/repositories"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"go.uber.org/zap"
	"google.golang.org/api/option"
)

// maxFCMTokensPerUser limits how many device tokens a user can register.
const maxFCMTokensPerUser = 10

// FCMService sends push notifications via Firebase Cloud Messaging.
// If credentials are not configured, the service is disabled (graceful degradation).
type FCMService struct {
	logger    *zap.Logger
	tokenRepo *repositories.FCMTokenRepo
	client    *messaging.Client
	enabled   bool
}

// NewFCMService initializes the Firebase app from a service account JSON file.
// If credentialsPath is empty or init fails, the service runs in disabled mode.
func NewFCMService(logger *zap.Logger, tokenRepo *repositories.FCMTokenRepo, credentialsPath string) *FCMService {
	s := &FCMService{logger: logger, tokenRepo: tokenRepo}

	if credentialsPath == "" {
		logger.Warn("FCM disabled: FCM_CREDENTIALS_PATH not set")
		return s
	}

	app, err := firebase.NewApp(context.Background(), nil, option.WithCredentialsFile(credentialsPath))
	if err != nil {
		logger.Error("FCM disabled: failed to init Firebase app", zap.Error(err))
		return s
	}

	client, err := app.Messaging(context.Background())
	if err != nil {
		logger.Error("FCM disabled: failed to get messaging client", zap.Error(err))
		return s
	}

	s.client = client
	s.enabled = true
	logger.Info("FCM initialized")
	return s
}

// IsEnabled returns whether FCM is configured and ready.
func (s *FCMService) IsEnabled() bool {
	return s.enabled
}

// Register stores an FCM token for a user, enforcing a per-user cap.
func (s *FCMService) Register(ctx context.Context, userID uint, token, platform string) error {
	count, err := s.tokenRepo.CountByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to check FCM token count: %w", err)
	}
	if count >= maxFCMTokensPerUser {
		return fmt.Errorf("FCM token limit reached (max %d)", maxFCMTokensPerUser)
	}

	return s.tokenRepo.Upsert(ctx, &models.FCMToken{
		UserID:   userID,
		Token:    token,
		Platform: platform,
	})
}

// Unregister removes an FCM token for a user.
func (s *FCMService) Unregister(ctx context.Context, userID uint, token string) error {
	return s.tokenRepo.DeleteByToken(ctx, userID, token)
}

// HasTokens checks if the user has any FCM tokens registered.
func (s *FCMService) HasTokens(ctx context.Context, userID uint) (bool, error) {
	count, err := s.tokenRepo.CountByUser(ctx, userID)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// SendNotification delivers a notification to every FCM token registered for recipientUserID.
// Invalid/unregistered tokens are pruned automatically.
func (s *FCMService) SendNotification(ctx context.Context, recipientUserID uint, payload PushPayload) error {
	if !s.enabled {
		return nil
	}

	tokens, err := s.tokenRepo.GetByUser(ctx, recipientUserID)
	if err != nil {
		return fmt.Errorf("failed to get FCM tokens: %w", err)
	}
	if len(tokens) == 0 {
		return nil
	}

	s.logger.Info("sending FCM notification",
		zap.Uint("recipient_id", recipientUserID),
		zap.Int("token_count", len(tokens)),
	)

	bgCtx := context.Background()
	for _, t := range tokens {
		go s.sendToToken(bgCtx, t, payload) //nolint:contextcheck // detached from caller
	}
	return nil
}

// sendToToken delivers to a single device token and prunes on permanent failure.
func (s *FCMService) sendToToken(ctx context.Context, t models.FCMToken, payload PushPayload) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("panic in FCM goroutine", zap.Any("panic", r))
		}
	}()

	msg := &messaging.Message{
		Token: t.Token,
		Notification: &messaging.Notification{
			Title: payload.Title,
			Body:  payload.Body,
		},
		Data: map[string]string{
			"chat_id": strconv.FormatUint(uint64(payload.ChatID), 10),
			"user_id": strconv.FormatUint(uint64(payload.UserID), 10),
			"tag":     payload.Tag,
		},
		Android: &messaging.AndroidConfig{
			Priority: "high",
			Notification: &messaging.AndroidNotification{
				Tag:         payload.Tag,
				ChannelID:   "messages",
				ClickAction: "FLUTTER_NOTIFICATION_CLICK",
			},
		},
	}

	if _, err := s.client.Send(ctx, msg); err != nil {
		if messaging.IsRegistrationTokenNotRegistered(err) || messaging.IsInvalidArgument(err) {
			s.logger.Info("FCM token invalid, removing",
				zap.Uint("user_id", t.UserID),
			)
			if delErr := s.tokenRepo.DeleteByTokenGlobal(ctx, t.Token); delErr != nil {
				s.logger.Error("failed to prune invalid FCM token", zap.Error(delErr))
			}
			return
		}
		s.logger.Warn("FCM send failed", zap.Error(err), zap.Uint("user_id", t.UserID))
		return
	}
	s.logger.Debug("FCM notification delivered", zap.Uint("user_id", t.UserID))
}
