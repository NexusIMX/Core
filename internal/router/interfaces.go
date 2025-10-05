package router

import (
	"context"
)

// RouteStorage defines the interface for managing user device routes
type RouteStorage interface {
	// RegisterRoute registers a device route for a user
	RegisterRoute(ctx context.Context, userID int64, deviceID, gatewayAddr string) error

	// UnregisterRoute removes a device route for a user
	UnregisterRoute(ctx context.Context, userID int64, deviceID string) error

	// GetRoute retrieves all device routes for a user
	GetRoute(ctx context.Context, userID int64) ([]*DeviceRoute, error)

	// KeepAlive updates the TTL for a user's device route
	KeepAlive(ctx context.Context, userID int64, deviceID string) error

	// GetOnlineStatus checks if a user is online and returns online device IDs
	GetOnlineStatus(ctx context.Context, userID int64) (bool, []string, error)
}
