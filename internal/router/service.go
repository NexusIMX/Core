package router

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dollarkillerx/im-system/pkg/logger"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	routeKeyPrefix    = "route:"
	presenceKeyPrefix = "presence:"
	defaultTTL        = 60 * time.Second
)

type DeviceRoute struct {
	DeviceID    string `json:"device_id"`
	GatewayAddr string `json:"gateway_addr"`
	LastActive  int64  `json:"last_active"`
}

type Service struct {
	redis *redis.Client
}

func NewService(redisClient *redis.Client) *Service {
	return &Service{
		redis: redisClient,
	}
}

// RegisterRoute registers a user's device route
func (s *Service) RegisterRoute(ctx context.Context, userID int64, deviceID, gatewayAddr string) error {
	routeKey := fmt.Sprintf("%s%d", routeKeyPrefix, userID)
	presenceKey := fmt.Sprintf("%s%d", presenceKeyPrefix, userID)

	route := &DeviceRoute{
		DeviceID:    deviceID,
		GatewayAddr: gatewayAddr,
		LastActive:  time.Now().Unix(),
	}

	routeData, err := json.Marshal(route)
	if err != nil {
		return fmt.Errorf("failed to marshal route: %w", err)
	}

	// Store route with TTL
	err = s.redis.HSet(ctx, routeKey, deviceID, routeData).Err()
	if err != nil {
		return fmt.Errorf("failed to set route: %w", err)
	}

	// Set TTL on route key
	s.redis.Expire(ctx, routeKey, defaultTTL)

	// Set presence to online
	s.redis.Set(ctx, presenceKey, "online", defaultTTL)

	logger.Log.Debug("Route registered",
		zap.Int64("user_id", userID),
		zap.String("device_id", deviceID),
		zap.String("gateway_addr", gatewayAddr),
	)

	return nil
}

// KeepAlive updates the TTL for a user's route
func (s *Service) KeepAlive(ctx context.Context, userID int64, deviceID string) error {
	routeKey := fmt.Sprintf("%s%d", routeKeyPrefix, userID)
	presenceKey := fmt.Sprintf("%s%d", presenceKeyPrefix, userID)

	// Check if device route exists
	exists, err := s.redis.HExists(ctx, routeKey, deviceID).Result()
	if err != nil {
		return fmt.Errorf("failed to check route existence: %w", err)
	}

	if !exists {
		return fmt.Errorf("route not found for device: %s", deviceID)
	}

	// Update last active time
	routeData, err := s.redis.HGet(ctx, routeKey, deviceID).Result()
	if err != nil {
		return fmt.Errorf("failed to get route: %w", err)
	}

	var route DeviceRoute
	if err := json.Unmarshal([]byte(routeData), &route); err != nil {
		return fmt.Errorf("failed to unmarshal route: %w", err)
	}

	route.LastActive = time.Now().Unix()
	updatedData, _ := json.Marshal(route)

	s.redis.HSet(ctx, routeKey, deviceID, updatedData)
	s.redis.Expire(ctx, routeKey, defaultTTL)
	s.redis.Expire(ctx, presenceKey, defaultTTL)

	return nil
}

// GetRoute retrieves all routes for a user
func (s *Service) GetRoute(ctx context.Context, userID int64) ([]*DeviceRoute, error) {
	routeKey := fmt.Sprintf("%s%d", routeKeyPrefix, userID)

	routes, err := s.redis.HGetAll(ctx, routeKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get routes: %w", err)
	}

	var deviceRoutes []*DeviceRoute
	for _, routeData := range routes {
		var route DeviceRoute
		if err := json.Unmarshal([]byte(routeData), &route); err != nil {
			logger.Log.Warn("Failed to unmarshal route", zap.Error(err))
			continue
		}
		deviceRoutes = append(deviceRoutes, &route)
	}

	return deviceRoutes, nil
}

// UnregisterRoute removes a user's device route
func (s *Service) UnregisterRoute(ctx context.Context, userID int64, deviceID string) error {
	routeKey := fmt.Sprintf("%s%d", routeKeyPrefix, userID)
	presenceKey := fmt.Sprintf("%s%d", presenceKeyPrefix, userID)

	// Remove device route
	err := s.redis.HDel(ctx, routeKey, deviceID).Err()
	if err != nil {
		return fmt.Errorf("failed to delete route: %w", err)
	}

	// Check if any devices remain
	count, err := s.redis.HLen(ctx, routeKey).Result()
	if err != nil {
		return fmt.Errorf("failed to check route count: %w", err)
	}

	// If no devices remain, set presence to offline
	if count == 0 {
		s.redis.Del(ctx, routeKey)
		s.redis.Set(ctx, presenceKey, "offline", defaultTTL)
	}

	logger.Log.Debug("Route unregistered",
		zap.Int64("user_id", userID),
		zap.String("device_id", deviceID),
	)

	return nil
}

// GetOnlineStatus checks if a user is online
func (s *Service) GetOnlineStatus(ctx context.Context, userID int64) (bool, []string, error) {
	routeKey := fmt.Sprintf("%s%d", routeKeyPrefix, userID)

	routes, err := s.redis.HKeys(ctx, routeKey).Result()
	if err != nil {
		return false, nil, fmt.Errorf("failed to get device IDs: %w", err)
	}

	online := len(routes) > 0
	return online, routes, nil
}
