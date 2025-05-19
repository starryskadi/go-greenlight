package main

import (
	"errors"
	"net/http"
	"time"

	"kyawzayarwin.com/greenlight/internal/data"
	"kyawzayarwin.com/greenlight/internal/validator"
)

func (app *application) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &input)

	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user := &data.User{
		Name:      input.Name,
		Email:     input.Email,
		Activated: false,
	}

	err = user.Password.Set(input.Password)

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	v := validator.New()

	if data.ValidateUser(v, user); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Users.Insert(user)

	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("email", "a user with this email addreess already existed")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}

		return
	}

	err = app.models.Permissions.AddForUser(user.ID, "movies:read")
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	token, err := app.models.Tokens.New(user.ID, 3*24*time.Hour, data.ScopeActivation)

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	app.background(func() {
		data := map[string]any{
			"activationToken": token.Plaintext,
			"userID":          user.ID,
		}

		err = app.mailer.Send(user.Email, "user_welcome.tmpl.html", data)

		if err != nil {
			app.logger.PrintError(err, nil)
			return
		}
	})

	err = app.writeJSON(w, 200, envelope{"users": user})

	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) activateUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		TokenPlainText string `json:"token"`
	}

	err := app.readJSON(w, r, &input)

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	v := validator.New()

	if data.ValidateTokenPlaintext(v, input.TokenPlainText); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	user, err := app.models.Users.GetFromToken(data.ScopeActivation, input.TokenPlainText)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			v.AddError("token", "invalid or expired activation token")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	user.Activated = true

	err = app.models.Users.Update(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.models.Tokens.DeleteAllForUser(data.ScopeActivation, user.ID)

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, 200, envelope{"user": user})

	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) createPasswordResetTokenHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email string 	`json:"email"`
	}  

	err := app.readJSON(w, r, &input)

	if err != nil {
		app.badRequestResponse(w, r, err)
		return 
	}

	v := validator.New() 

	data.ValidateEmail(v, input.Email)

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return 
	}

	user, err := app.models.Users.GetByEmail(input.Email)

	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.badRequestResponse(w, r, err)
			return 
		default: 
			app.serverErrorResponse(w, r, err)
		}
		return 
	}

	if !user.Activated {
		v.AddError("email", "user account must be activated")
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	token, err := app.models.Tokens.New(user.ID, 15 * time.Minute, data.ScopePasswordReset)

	app.background(func() {
		data := map[string]interface{}{
			"passwordResetToken": token.Plaintext,
		}

		err := app.mailer.Send(user.Email, "password_rest.tmpl.html", data) 

		if err != nil {
			app.logger.PrintError(err, nil)
		}
	})

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, 200, envelope{ "message": "an email will be sent to you containing password reset instructions" })

	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) updateUserPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Token string 	`json:"token"`
		Password string	`json:"password"`
	}

	err := app.readJSON(w, r, &input)

	if err != nil {
		app.badRequestResponse(w, r, err)
		return 
	}

	v := validator.New()

	data.ValidateTokenPlaintext(v, input.Token)

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return 
	}

	user, err := app.models.Users.GetFromToken(data.ScopePasswordReset, input.Token);

	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			v.AddError("token", "invalid or expired password reset token")
			app.failedValidationResponse(w, r, v.Errors)
		default: 
			app.serverErrorResponse(w, r, err)
		}
		return 
	}

	err = user.Password.Set(input.Password)
 
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return 
	}

	err = app.models.Users.Update(user)

	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return 
	}

	err = app.models.Tokens.DeleteAllForUser(data.ScopePasswordReset, user.ID)

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return 
	}

	err = app.writeJSON(w, http.StatusOK, envelope{
		"message": "your password was successfully reset",
	})

	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}