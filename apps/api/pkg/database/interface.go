package database

import (
	"context"

	"github.com/unkeyed/unkey/apps/api/pkg/entities"
)

type Database interface {
	CreateApi(ctx context.Context, newApi entities.Api) error
	GetApi(ctx context.Context, apiId string) (entities.Api, error)
	GetApiByKeyAuthId(ctx context.Context, keyAuthId string) (entities.Api, error)

	CreateKey(ctx context.Context, newKey entities.Key) error
	UpdateKey(ctx context.Context, key entities.Key) error

	DeleteKey(ctx context.Context, keyId string) error
	GetKeyByHash(ctx context.Context, hash string) (entities.Key, error)
	GetKeyById(ctx context.Context, keyId string) (entities.Key, error)
	CountKeys(ctx context.Context, keyAuthId string) (int, error)
	ListKeysByKeyAuthId(ctx context.Context, keyAuthId string, limit int, offset int, ownerId string) ([]entities.Key, error)
	CreateWorkspace(ctx context.Context, newWorkspace entities.Workspace) error

	CreateKeyAuth(ctx context.Context, newKeyAuth entities.KeyAuth) error
	GetKeyAuth(ctx context.Context, keyAuthId string) (entities.KeyAuth, error)

	GetWorkspace(ctx context.Context, workspaceId string) (entities.Workspace, error)
	DecrementRemainingKeyUsage(ctx context.Context, keyId string) (int64, error)
}
