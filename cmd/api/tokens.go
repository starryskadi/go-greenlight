package main

import (
	"errors"
	"net/http"
	"time"

	"kyawzayarwin.com/greenlight/internal/data"
	"kyawzayarwin.com/greenlight/internal/validator"
)

func (app *application) createAuthenticationTokenHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &input)

	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()

	data.ValidateEmail(v, input.Email)
	data.ValidatePasswordPlaintext(v, input.Password)

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	user, err := app.models.Users.GetByEmail(input.Email)

	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.invalidCredentialsResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	match, err := user.Password.Matches(input.Password)

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if !match {
		app.invalidCredentialsResponse(w, r)
		return
	}

	token, err := app.models.Tokens.New(user.ID, 60*time.Minute, data.ScopeAuthentication)

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"authentication_token": token})

	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

}

func (app *application) createActivationTokenHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email string	`json:"email"`
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

	token, err := app.models.Tokens.New(user.ID, 3 * 24 * time.Hour, data.ScopeActivation)

	app.background(func() {
		data := map[string]interface{}{
			"activationToken": token.Plaintext,
		}

		err := app.mailer.Send(user.Email, "token_activation.tmpl.html", data) 

		if err != nil {
			app.logger.PrintError(err, nil)
		}
	})

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, 200, envelope{ "message": "an email will be sent to you containing activation token" })

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