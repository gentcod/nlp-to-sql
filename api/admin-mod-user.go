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
	userId := ctx.Param("userId")

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

	_, err = server.store.GetUser(ctx, uuid.MustParse(userId))
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
		ID:        uuid.MustParse(userId),
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
	userId := ctx.Param("userId")

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

	user, err := server.store.GetUser(ctx, uuid.MustParse(userId))
	if err != nil {
		if err == sql.ErrNoRows {
			msg := "user account not found"
			ctx.JSON(http.StatusNotFound, apiErrorResponse(errors.New(msg)))
			return
		}
		ctx.JSON(http.StatusInternalServerError, apiErrorResponse(err))
		return
	}

	err = server.store.DeleteUserTx(ctx, uuid.MustParse(userId), user.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			msg := "user authentication not found"
			ctx.JSON(http.StatusNotFound, apiErrorResponse(errors.New(msg)))
			return
		}
		ctx.JSON(http.StatusInternalServerError, apiErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, apiServerResponse("Deleted user successfully.", ""))
}
