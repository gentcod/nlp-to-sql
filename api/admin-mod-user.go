package api

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	db "github.com/gentcod/nlp-to-sql/internal/database"
	"github.com/gentcod/nlp-to-sql/token"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (server *Server) adminRestrictUser(ctx *gin.Context) {
	var req adminModUserRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, apiErrorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	_, err := server.store.GetAuth(ctx, authPayload.UserID)
	if err != nil {
		if err == sql.ErrNoRows {
			msg := "admin not found"
			ctx.JSON(http.StatusUnauthorized, apiErrorResponse(errors.New(msg)))
			return
		}
		ctx.JSON(http.StatusInternalServerError, apiErrorResponse(err))
		return
	}

	_, err = server.store.GetUser(ctx, uuid.MustParse(req.userId))
	if err != nil {
		if err == sql.ErrNoRows {
			msg := "user not found"
			ctx.JSON(http.StatusNotFound, apiErrorResponse(errors.New(msg)))
			return
		}
		ctx.JSON(http.StatusInternalServerError, apiErrorResponse(err))
		return
	}

	err = server.store.RestrictAuth(ctx, db.RestrictAuthParams{
		ID: uuid.MustParse(req.userId),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		if err == sql.ErrNoRows {
			msg := "user authentication not found"
			ctx.JSON(http.StatusNotFound, apiErrorResponse(errors.New(msg)))
			return
		}
		ctx.JSON(http.StatusInternalServerError, apiErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, apiServerResponse("Restricted user successfully", ""))
}

func (server *Server) adminDeleteUser(ctx *gin.Context) {
	var req adminModUserRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, apiErrorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	_, err := server.store.GetAuth(ctx, authPayload.UserID)
	if err != nil {
		if err == sql.ErrNoRows {
			msg := "admin not found"
			ctx.JSON(http.StatusUnauthorized, apiErrorResponse(errors.New(msg)))
			return
		}
		ctx.JSON(http.StatusInternalServerError, apiErrorResponse(err))
		return
	}

	user, err := server.store.GetUser(ctx, uuid.MustParse(req.userId))
	if err != nil {
		if err == sql.ErrNoRows {
			msg := "user account not found"
			ctx.JSON(http.StatusNotFound, apiErrorResponse(errors.New(msg)))
			return
		}
		ctx.JSON(http.StatusInternalServerError, apiErrorResponse(err))
		return
	}

	err = server.store.DeleteUserTx(ctx, uuid.MustParse(req.userId), user.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			msg := "user authentication not found"
			ctx.JSON(http.StatusNotFound, apiErrorResponse(errors.New(msg)))
			return
		}
		ctx.JSON(http.StatusInternalServerError, apiErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, apiServerResponse("Restricted user successfully", ""))
}
