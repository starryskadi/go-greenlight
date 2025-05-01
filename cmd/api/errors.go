package main

import (
	"net/http"
)

func (app *application) logError(err error) {
	app.logger.Println(err)
}	

func (app *application) errorResponse(w http.ResponseWriter, status int, message any) {
	env := envelope{"error": message}

	err := app.writeJSON(w, status, env)

	if err != nil {
		app.logError(err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (app *application) serverErrorResponse(w http.ResponseWriter, err error) {
	app.logError(err)

	msg := "the server encountered a problem and could not process your request"

	app.errorResponse(w, http.StatusInternalServerError, msg)
}

func (app *application) failedValidationResponse(w http.ResponseWriter, errors map[string]string) {
	app.errorResponse(w, http.StatusUnprocessableEntity, errors)
}