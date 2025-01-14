package server

import (
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/unkeyed/unkey/apps/api/pkg/database"
	"github.com/unkeyed/unkey/apps/api/pkg/entities"
	"github.com/unkeyed/unkey/apps/api/pkg/hash"
	"github.com/unkeyed/unkey/apps/api/pkg/kafka"
	"github.com/unkeyed/unkey/apps/api/pkg/keys"
	"github.com/unkeyed/unkey/apps/api/pkg/uid"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type CreateKeyRequest struct {
	ApiId      string         `json:"apiId" validate:"required"`
	Prefix     string         `json:"prefix"`
	Name       string         `json:"name"`
	ByteLength int            `json:"byteLength"`
	OwnerId    string         `json:"ownerId"`
	Meta       map[string]any `json:"meta"`
	Expires    int64          `json:"expires"`
	Ratelimit  *struct {
		Type           string `json:"type"`
		Limit          int64  `json:"limit"`
		RefillRate     int64  `json:"refillRate"`
		RefillInterval int64  `json:"refillInterval"`
	} `json:"ratelimit"`
	// ForWorkspaceId is used internally when the frontend wants to create a new root key.
	// Therefore we might not want to add this field to our docs.
	ForWorkspaceId string `json:"forWorkspaceId"`

	// How often this key may be used
	// `undefined`, `0` or negative to disable
	Remaining int64 `json:"remaining,omitempty"`
}

type CreateKeyResponse struct {
	Key   string `json:"key"`
	KeyId string `json:"keyId"`
}

func (s *Server) createKey(c *fiber.Ctx) error {
	ctx, span := s.tracer.Start(c.UserContext(), "server.createKey")
	defer span.End()

	req := CreateKeyRequest{
		// These act as default
		ByteLength: 16,
	}
	err := c.BodyParser(&req)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Code:  BAD_REQUEST,
			Error: fmt.Sprintf("unable to parse body: %s", err.Error()),
		})
	}

	err = s.validator.Struct(req)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Code:  BAD_REQUEST,
			Error: fmt.Sprintf("unable to validate body: %s", err.Error()),
		})
	}

	if req.Expires > 0 && req.Expires < time.Now().UnixMilli() {
		return c.Status(http.StatusBadRequest).JSON(
			ErrorResponse{
				Code:  BAD_REQUEST,
				Error: "'expires' must be in the future, did you pass in a timestamp in seconds instead of milliseconds?",
			})
	}

	authHash, err := getKeyHash(c.Get("Authorization"))
	if err != nil {
		return err
	}

	authKey, err := s.db.GetKeyByHash(ctx, authHash)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return c.Status(http.StatusUnauthorized).JSON(ErrorResponse{
				Code:  UNAUTHORIZED,
				Error: "unauthorized",
			})
		}

		return c.Status(http.StatusInternalServerError).JSON(ErrorResponse{
			Code:  INTERNAL_SERVER_ERROR,
			Error: fmt.Sprintf("unable to find key: %s", err.Error()),
		})
	}

	if authKey.ForWorkspaceId == "" {
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Code:  BAD_REQUEST,
			Error: "wrong key type",
		})
	}

	api, err := s.db.GetApi(ctx, req.ApiId)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
				Code:  BAD_REQUEST,
				Error: "wrong apiId",
			})
		}
		return c.Status(http.StatusInternalServerError).JSON(ErrorResponse{
			Code:  INTERNAL_SERVER_ERROR,
			Error: fmt.Sprintf("unable to find api: %s", err.Error()),
		})
	}
	if api.WorkspaceId != authKey.ForWorkspaceId {
		return c.Status(http.StatusUnauthorized).JSON(ErrorResponse{
			Code:  UNAUTHORIZED,
			Error: "access to workspace denied",
		})
	}

	if api.AuthType != entities.AuthTypeKey || api.KeyAuthId == "" {
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Code:  BAD_REQUEST,
			Error: fmt.Sprintf("api is not set up to handle key auth: %+v", api),
		})
	}

	keyValue, err := keys.NewV1Key(req.Prefix, req.ByteLength)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(ErrorResponse{
			Code:  INTERNAL_SERVER_ERROR,
			Error: err.Error(),
		})
	}
	// how many chars to store, this includes the prefix, delimiter and the first 4 characters of the key
	startLength := len(req.Prefix) + 5
	keyHash := hash.Sha256(keyValue)

	newKey := entities.Key{
		Id:          uid.Key(),
		KeyAuthId:   api.KeyAuthId,
		WorkspaceId: authKey.ForWorkspaceId,
		Name:        req.Name,
		Hash:        keyHash,
		Start:       keyValue[:startLength],
		OwnerId:     req.OwnerId,
		Meta:        req.Meta,
		CreatedAt:   time.Now(),
	}
	if req.Expires > 0 {
		newKey.Expires = time.UnixMilli(req.Expires)
	}
	if req.Remaining > 0 {
		newKey.Remaining.Enabled = true
		newKey.Remaining.Remaining = req.Remaining
	}
	if req.Ratelimit != nil {
		newKey.Ratelimit = &entities.Ratelimit{
			Type:           req.Ratelimit.Type,
			Limit:          req.Ratelimit.Limit,
			RefillRate:     req.Ratelimit.RefillRate,
			RefillInterval: req.Ratelimit.RefillInterval,
		}
	}

	err = s.db.CreateKey(ctx, newKey)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(ErrorResponse{
			Code:  INTERNAL_SERVER_ERROR,
			Error: fmt.Sprintf("unable to store key: %s", err.Error()),
		})
	}
	if s.kafka != nil {

		go func() {
			err := s.kafka.ProduceKeyEvent(ctx, kafka.KeyCreated, newKey.Id, newKey.Hash)
			if err != nil {
				s.logger.Error("unable to emit new key event to kafka", zap.Error(err))
			}
		}()
	}

	return c.JSON(CreateKeyResponse{
		Key:   keyValue,
		KeyId: newKey.Id,
	})
}
