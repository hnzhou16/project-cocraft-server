package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/hnzhou16/project-social/internal/mailer"
	"github.com/hnzhou16/project-social/internal/security"
	"github.com/hnzhou16/project-social/internal/storage"
)

type RegisterUserPayload struct {
	Username string          `json:"username" validate:"required"`
	Email    string          `json:"email" validate:"required,email,valid_email"`
	Password string          `json:"password" validate:"required,valid_password"`
	Role     security.Role   `json:"role" validate:"required,valid_role"`
	Profile  storage.Profile `json:"profile"`
	Rating   storage.Rating  `json:"rating,omitempty"`
}

type CreateTokenPayload struct {
	Email    string `json:"email" validate:"required,email,valid_email"`
	Password string `json:"password" validate:"required,valid_password"`
}

type UserWithToken struct {
	*storage.User
	Token string `json:"token"`
}

func (app *application) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	var payload RegisterUserPayload
	if err := ReadJSON(w, r, &payload); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	hashedPassword, err := security.HashPassword(payload.Password)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	user := &storage.User{
		Username: payload.Username,
		Email:    payload.Email,
		Password: hashedPassword,
		Role:     payload.Role,
		Profile:  payload.Profile,
		Rating:   payload.Rating,
	}

	ctx := r.Context()

	// plainToken sent to client in the email
	plainToken := uuid.New().String()

	// hash token to store in db
	hash := sha256.Sum256([]byte(plainToken))
	hashToken := hex.EncodeToString(hash[:])

	err = app.storage.User.CreateAndInvite(ctx, user, hashToken, app.config.mailConfig.exp)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrDupUsername):
			app.conflictError(w, r, "DUPLICATE_USERNAME", err)
		case errors.Is(err, storage.ErrDupEmail):
			app.conflictError(w, r, "DUPLICATE_EMAIL", err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	userWithToken := &UserWithToken{
		User:  user,
		Token: plainToken,
	}

	// send email
	activationURL := fmt.Sprintf("%s/%s", app.config.mailConfig.activationURL, plainToken)
	displayUsername := strings.ReplaceAll(user.Username, "_", " ")

	activationData := struct {
		Username      string
		ActivationURL string
	}{
		Username:      displayUsername,
		ActivationURL: activationURL,
	}

	status, err := app.mailer.Send(mailer.UserActivateTemplate, displayUsername, user.Email, activationData)
	if err != nil {
		app.logger.Errorw("error sending activation email", "error", err)

		// rollback user creation if email fails (SAGA pattern)
		if err := app.storage.User.Delete(ctx, user.ID); err != nil {
			app.logger.Errorw("error deleting user", "error", err)
		}

		app.internalServerError(w, r, err)
		return
	}
	app.logger.Infow("Email sent", "status code", status)

	app.OutputJSON(w, http.StatusCreated, userWithToken)
}

func (app *application) createTokenHandler(w http.ResponseWriter, r *http.Request) {
	var payload CreateTokenPayload
	if err := ReadJSON(w, r, &payload); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	user, err := app.storage.User.GetByEmail(r.Context(), payload.Email)
	if err != nil {
		// return unauthorized error for all scenarios to avoid enumeration attack
		switch {
		case errors.Is(err, storage.ErrUserNotFound):
			app.unauthorizedError(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	if err := security.VerifyPassword(user.Password, payload.Password); err != nil {
		app.unauthorizedError(w, r, err)
		return
	}

	claims := jwt.MapClaims{
		"sub": user.ID, // sub – subject of JWT (the user)
		"exp": time.Now().Add(app.config.authConfig.exp).Unix(),
		"iat": time.Now().Unix(),
		"nbf": time.Now().Unix(), // nbf – not before time JWT can be accepted
		"iss": app.config.authConfig.iss,
		"aud": app.config.authConfig.iss, // aud – audience/recipient for JWT (the issuer)
	}

	token, err := app.authenticator.GenerateToken(claims)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.OutputJSON(w, http.StatusCreated, token)
}
