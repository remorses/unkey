package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/unkeyed/unkey/apps/api/pkg/database/models"
	"github.com/unkeyed/unkey/apps/api/pkg/entities"
)

func keyModelToEntity(model *models.Key) (entities.Key, error) {

	key := entities.Key{}
	key.Id = model.ID

	if model.KeyAuthID.Valid {
		key.KeyAuthId = model.KeyAuthID.String
	}
	key.WorkspaceId = model.WorkspaceID
	key.Hash = model.Hash
	key.Start = model.Start

	key.CreatedAt = model.CreatedAt

	if model.OwnerID.Valid {
		key.OwnerId = model.OwnerID.String
	}

	if model.Name.Valid {
		key.Name = model.Name.String
	}

	if model.Expires.Valid {
		key.Expires = model.Expires.Time
	}

	if model.ForWorkspaceID.Valid {
		key.ForWorkspaceId = model.ForWorkspaceID.String
	}

	if model.Meta.Valid {
		err := json.Unmarshal([]byte(model.Meta.String), &key.Meta)
		if err != nil {
			return entities.Key{}, fmt.Errorf("unable to unmarshal meta: %w", err)
		}
	}

	if model.RatelimitType.Valid {
		key.Ratelimit = &entities.Ratelimit{
			Type:           model.RatelimitType.String,
			Limit:          model.RatelimitLimit.Int64,
			RefillRate:     model.RatelimitRefillRate.Int64,
			RefillInterval: model.RatelimitRefillInterval.Int64,
		}
	}

	if model.RemainingRequests.Valid {
		key.Remaining.Enabled = true
		key.Remaining.Remaining = model.RemainingRequests.Int64
	}
	return key, nil
}

func keyEntityToModel(e entities.Key) (*models.Key, error) {
	metaBuf, err := json.Marshal(e.Meta)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal meta: %w", err)
	}

	key := &models.Key{
		ID:          e.Id,
		KeyAuthID:   sql.NullString{String: e.KeyAuthId, Valid: e.KeyAuthId != ""},
		Start:       e.Start,
		WorkspaceID: e.WorkspaceId,
		Hash:        e.Hash,
		OwnerID: sql.NullString{
			String: e.OwnerId,
			Valid:  e.OwnerId != "",
		},
		Name: sql.NullString{
			String: e.Name,
			Valid:  e.Name != "",
		},
		Meta:      sql.NullString{String: string(metaBuf), Valid: len(metaBuf) > 0},
		CreatedAt: e.CreatedAt,
		Expires: sql.NullTime{
			Time:  e.Expires,
			Valid: !e.Expires.IsZero(),
		},

		ForWorkspaceID: sql.NullString{String: e.ForWorkspaceId, Valid: e.ForWorkspaceId != ""},
	}
	if e.Ratelimit != nil {
		key.RatelimitType = sql.NullString{String: e.Ratelimit.Type, Valid: e.Ratelimit.Type != ""}
		key.RatelimitLimit = sql.NullInt64{Int64: e.Ratelimit.Limit, Valid: e.Ratelimit.Limit > 0}
		key.RatelimitRefillRate = sql.NullInt64{Int64: e.Ratelimit.RefillRate, Valid: e.Ratelimit.RefillRate > 0}
		key.RatelimitRefillInterval = sql.NullInt64{Int64: e.Ratelimit.RefillInterval, Valid: e.Ratelimit.RefillRate > 0}
	}

	if e.Remaining.Enabled {
		key.RemainingRequests = sql.NullInt64{Int64: e.Remaining.Remaining, Valid: true}
	}

	return key, nil

}

func workspaceEntityToModel(w entities.Workspace) *models.Workspace {

	return &models.Workspace{
		ID:       w.Id,
		Name:     w.Name,
		Slug:     w.Slug,
		TenantID: w.TenantId,
		Internal: w.Internal,
	}

}

func workspaceModelToEntity(model *models.Workspace) entities.Workspace {
	return entities.Workspace{
		Id:       model.ID,
		Name:     model.Name,
		Slug:     model.Slug,
		TenantId: model.TenantID,
		Internal: model.Internal,
	}

}

func apiEntityToModel(a entities.Api) *models.API {

	return &models.API{
		ID:          a.Id,
		Name:        a.Name,
		WorkspaceID: a.WorkspaceId,
		IPWhitelist: sql.NullString{String: strings.Join(a.IpWhitelist, ","), Valid: len(a.IpWhitelist) > 0},
		AuthType:    models.NullAuthType{AuthType: 1, Valid: true},
		KeyAuthID:   sql.NullString{String: a.KeyAuthId, Valid: a.KeyAuthId != ""},
	}

}

func apiModelToEntity(model *models.API) entities.Api {
	a := entities.Api{
		Id:          model.ID,
		Name:        model.Name,
		WorkspaceId: model.WorkspaceID,
		KeyAuthId:   model.KeyAuthID.String,
	}

	if model.IPWhitelist.Valid {
		a.IpWhitelist = strings.Split(model.IPWhitelist.String, ",")
	}

	if model.AuthType.Valid {
		switch model.AuthType.AuthType.String() {
		case "key":
			a.KeyAuthId = model.KeyAuthID.String
			a.AuthType = entities.AuthTypeKey
		case "jwt":
			a.AuthType = entities.AuthTypeJWT
		}

	}

	return a

}

func keyAuthEntityToModel(a entities.KeyAuth) *models.KeyAuth {

	return &models.KeyAuth{
		ID:          a.Id,
		WorkspaceID: a.WorkspaceId,
	}

}

func keyAuthModelToEntity(model *models.KeyAuth) entities.KeyAuth {
	a := entities.KeyAuth{
		Id:          model.ID,
		WorkspaceId: model.WorkspaceID,
	}

	return a

}
