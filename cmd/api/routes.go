package main

import (
	"expvar"
	"net/http"

	"kyawzayarwin.com/greenlight/internal/data"
)

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /v1/healthcheck", app.healthCheckHandler)

	protectedRoutes := CreateMiddlewareStack(
		app.requireActivateUser,
	)

	// movies handler
	mux.Handle("GET /v1/movies", protectedRoutes(app.requirePermission(data.PermissionMovieRead, http.HandlerFunc(app.listMoviesHandler))))
	mux.Handle("POST /v1/movies", protectedRoutes(app.requirePermission(data.PermissionMovieWrite, http.HandlerFunc(app.createMovieHandler))))
	mux.Handle("GET /v1/movies/{id}", protectedRoutes(app.requirePermission(data.PermissionMovieRead, http.HandlerFunc(app.showMovieHandler))))
	mux.Handle("PATCH /v1/movies/{id}", protectedRoutes(app.requirePermission(data.PermissionMovieWrite, http.HandlerFunc(app.updateMovieHandler))))
	mux.Handle("DELETE /v1/movies/{id}", protectedRoutes(app.requirePermission(data.PermissionMovieWrite, http.HandlerFunc(app.deleteMovieHandler))))

	// Users Handlers
	mux.HandleFunc("POST /v1/users", app.registerUserHandler)
	mux.HandleFunc("PUT /v1/users/activated", app.activateUserHandler)
	mux.HandleFunc("POST /v1/tokens/authentication", app.createAuthenticationTokenHandler)
	mux.HandleFunc("POST /v1/tokens/password-reset", app.createPasswordResetTokenHandler)
	mux.HandleFunc("PUT /v1/users/password", app.updateUserPasswordHandler)

	// Debug Handlers
	mux.Handle("GET /debug/vars", expvar.Handler())

	defaultMiddleWare := CreateMiddlewareStack(
		app.metrics,
		app.recoverPanic,
		app.enableCORS,
		app.rateLimit,
		app.authenticate,
	)

	return defaultMiddleWare(mux)
}
