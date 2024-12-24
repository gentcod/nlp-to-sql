package api

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	db "github.com/gentcod/nlp-to-sql/internal/database"
	"github.com/gentcod/nlp-to-sql/token"
	"github.com/gentcod/nlp-to-sql/util"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// TODO: Implement password confirmation when updating account and password auth for getting account
func (server *Server) createUser(ctx *gin.Context) {
	var req createUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, apiErrorResponse(err))
		return
	}

	harshedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, apiErrorResponse(err))
		return
	}

	arg := db.CreateUserTxParams{
		CreateAuthParams: db.CreateAuthParams{
			ID:              uuid.New(),
			Email:           req.Email,
			HarshedPassword: harshedPassword,
		},
		CreateUserParams: db.CreateUserParams{
			ID:       uuid.New(),
			Username: req.Username,
			FullName: req.FullName,
		},
	}

	usertx, err := server.store.CreateUserTx(ctx, arg)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				msg := "user email/usenname already exists"
				ctx.JSON(http.StatusForbidden, apiErrorResponse(errors.New(msg)))
				return
			}
		}
		ctx.JSON(http.StatusInternalServerError, apiErrorResponse(err))
		return
	}

	profile := getUserProfile(usertx)

	ctx.JSON(http.StatusOK, apiServerResponse("user account created sucessfully", profile))
}

func (server *Server) updateUser(ctx *gin.Context) {
	var req updateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, apiErrorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	auth, err := server.store.GetAuth(ctx, authPayload.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, apiErrorResponse(err))
		return
	}

	user, err := server.store.GetUser(ctx, auth.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, apiErrorResponse(err))
		return
	}

	var newHarshedPassword string
	if req.Password != "" {
		newHarshedPassword, err = util.HashPassword(req.Password)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, apiErrorResponse(err))
			return
		}
	}

	txArg := db.UpdateUserTxParams{
		UpdateAuthParams: db.UpdateAuthParams{
			ID: auth.ID,
			Email: sql.NullString{
				String: req.Email,
				Valid:  req.Email != "",
			},
			HarshedPassword: sql.NullString{
				String: newHarshedPassword,
				Valid:  newHarshedPassword != "",
			},
		},
		UpdateUserParams: db.UpdateUserParams{
			ID: user.ID,
			Username: sql.NullString{
				String: req.Username,
				Valid:  req.Username != "",
			},
			FullName: sql.NullString{
				String: req.FullName,
				Valid:  req.FullName != "",
			},
		},
	}

	txArg.UpdateAuthParams.PasswordChangedAt = sql.NullTime{
		Time:  time.Now(),
		Valid: newHarshedPassword != "",
	}

	updateTx, err := server.store.UpdateUserTx(ctx, txArg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, apiErrorResponse(err))
		return
	}

	profile := getUserProfile(updateTx)
	ctx.JSON(http.StatusOK, apiServerResponse("user account updated sucessfully", profile))
}

func (server *Server) deleteUser(ctx *gin.Context) {
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	auth, err := server.store.GetAuth(ctx, authPayload.UserID)
	if err != nil {
		if err == sql.ErrNoRows {
			msg := "user not found"
			ctx.JSON(http.StatusUnauthorized, apiErrorResponse(errors.New(msg)))
			return
		}
		ctx.JSON(http.StatusInternalServerError, apiErrorResponse(err))
		return
	}

	user, err := server.store.GetUser(ctx, auth.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			msg := "user account not found"
			ctx.JSON(http.StatusNotFound, apiErrorResponse(errors.New(msg)))
			return
		}
		ctx.JSON(http.StatusInternalServerError, apiErrorResponse(err))
		return
	}

	err = server.store.DeleteUserTx(ctx, auth.ID, user.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			msg := "user authentication not found"
			ctx.JSON(http.StatusNotFound, apiErrorResponse(errors.New(msg)))
			return
		}
		ctx.JSON(http.StatusInternalServerError, apiErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, apiServerResponse("Deleted account successfully", ""))
}

func (server *Server) loginUser(ctx *gin.Context) {
	var req loginUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, apiErrorResponse(err))
		return
	}

	auth, valid := server.validateUser(ctx, req.Email, req.Password)
	if !valid {
		return
	}

	user, err := server.store.GetUser(ctx, auth.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, apiErrorResponse(err))
		return
	}

	accessToken, accessPayload, err := server.tokenGenerator.CreateToken(user.Username, auth.ID, server.config.AccessTokenDuration)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, apiErrorResponse(err))
		return
	}

	resp := loginUserResponse{
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessPayload.ExpiredAt,
		User: UserProfile{
			Username:          user.Username,
			FullName:          user.FullName,
			Email:             auth.Email,
			CreatedAt:         user.CreatedAt,
			PasswordChangedAt: auth.PasswordChangedAt,
		},
	}

	ctx.JSON(http.StatusOK, apiServerResponse("Account login sucessfully", resp))
}

func (server *Server) validateUser(ctx *gin.Context, email string, password string) (db.Auth, bool) {
	auth, err := server.store.ValidateAuth(ctx, email)
	if err != nil {
		if err == sql.ErrNoRows {
			msg := "user not found"
			ctx.JSON(http.StatusNotFound, apiErrorResponse(errors.New(msg)))
			return auth, false
		}
		ctx.JSON(http.StatusInternalServerError, apiErrorResponse(err))
		return auth, false
	}

	if auth.Role.RoleType != db.RoleTypeUser {
		msg := "Invalid route."
		ctx.JSON(http.StatusUnauthorized, apiErrorResponse(errors.New(msg)))
		return auth, false
	}

	if auth.Restricted {
		msg := "Account has been restricted."
		ctx.JSON(http.StatusUnauthorized, apiErrorResponse(errors.New(msg)))
		return auth, false
	}

	if auth.Deleted {
		msg := "Account no longer exists in our records. Attempt account recovery"
		ctx.JSON(http.StatusUnauthorized, apiErrorResponse(errors.New(msg)))
		return auth, false
	}

	err = util.CheckPassword(password, auth.HarshedPassword)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, apiErrorResponse(err))
		return auth, false
	}

	return auth, true
}

func getUserProfile(usertx db.UserTxResult) UserProfile {
	return UserProfile{
		Username:          usertx.User.Username,
		FullName:          usertx.User.FullName,
		Email:             usertx.Auth.Email,
		CreatedAt:         usertx.User.CreatedAt,
		PasswordChangedAt: usertx.Auth.PasswordChangedAt,
	}
}

// TODO: implement logic for validating difference for password, username and full_name
// func validateUpdateReq
// _, valid := server.validateUser(ctx, auth.Email, req.Password)
// if valid {
// 	msg := "current password and new password must not be the same"
// 	ctx.JSON(http.StatusBadRequest, apiErrorResponse(errors.New(msg)))
// 	return
// }

// TODO: Implement logic account recovery after deletion or restriction.
