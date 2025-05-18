package main

import (
	"errors"
	"expvar"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/felixge/httpsnoop"
	"golang.org/x/time/rate"
	"kyawzayarwin.com/greenlight/internal/data"
	"kyawzayarwin.com/greenlight/internal/validator"
)

type Middleware func(http.Handler) http.Handler

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				app.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
} 

func (app *application) rateLimit(next http.Handler) http.Handler {
		type client struct {
			limiter 	*rate.Limiter
			lastSeen 	time.Time
		}

		var (
			mu sync.Mutex
			clients = make(map[string]*client)
		)

		go func() {
			for {
				time.Sleep(time.Minute)

				mu.Lock()

				for ip, client := range clients {
					if time.Since(client.lastSeen) > 3*time.Minute {
						delete(clients, ip)
					}
				}

				mu.Unlock()
			}
		}()
		
	
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !app.config.limiter.enabled {
				ip, _, err := net.SplitHostPort(r.RemoteAddr)
				if err != nil {
					app.serverErrorResponse(w, r, err)
					return
				}

				mu.Lock()

				if _, found := clients[ip]; !found {
					clients[ip] = &client{limiter: rate.NewLimiter(rate.Limit(app.config.limiter.rps), app.config.limiter.burst)}
				}

				clients[ip].lastSeen = time.Now()

				if !clients[ip].limiter.Allow() {
					mu.Unlock()
					app.rateLimitExceededResponse(w, r)
					return
				}

				mu.Unlock()
			}
			

			next.ServeHTTP(w, r)
		})
}

func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Authorization")

		authroziationHeader := r.Header.Get("Authorization")

		if authroziationHeader == "" {
			r = app.ContextSetUser(r, data.AnonymousUser)
			next.ServeHTTP(w ,r)
			return 
		}

		headerParts := strings.Split(authroziationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		} 

		token := headerParts[1]

		v := validator.New()

		data.ValidateTokenPlaintext(v, token)

		if !v.Valid() {
			app.invalidAuthenticationTokenResponse(w, r)
			return 
		}

		user, err := app.models.Users.GetFromToken(data.ScopeAuthentication, token)

		if err != nil {
			switch {
			case errors.Is(err, data.ErrRecordNotFound):
				app.invalidAuthenticationTokenResponse(w, r)
			default:
				app.serverErrorResponse(w, r, err)
			}
			return 
		}

		r = app.ContextSetUser(r, user)
		next.ServeHTTP(w, r)
	})
} 

func (app *application) requireAuthenticatedUser(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.ContextGetUser(r)
		if user.IsAnonymous() {
			app.authenticationRequiredResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (app *application) requireActivateUser(next http.Handler) http.Handler {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.ContextGetUser(r)

		if user.IsAnonymous() {
			app.authenticationRequiredResponse(w, r)
			return 
		}

		if !user.Activated {
			app.inactiveAccountResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})

	return app.requireAuthenticatedUser(fn)
}

func (app *application) requirePermission(code string, next http.Handler) http.Handler {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.ContextGetUser(r)

		permissions, err := app.models.Permissions.GetAllForUser(user.ID)

		if err != nil {
			app.serverErrorResponse(w, r, err)
			return 
		}
		 
	    if !permissions.Include(code) {
			app.notPermittedResponse(w, r)
			return 
		}

		next.ServeHTTP(w, r)
	})

	return app.requireActivateUser(fn)
} 

func (app *application) enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Vary", "Origin")

		origin := r.Header.Get("Origin")

		if origin != "" {
			for i := range app.config.cors.trustedOrigin {
				if origin == app.config.cors.trustedOrigin[i] {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					break
				}

				// Handle preflight request
				if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
					w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, PUT, PATCH, DELETE")
					w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
					w.Header().Set("Access-Control-Max-Age", "60")

					w.WriteHeader(http.StatusOK)
					return 
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (app *application) metrics(next http.Handler) http.Handler {
	totalRequestsReceived := expvar.NewInt("total_requests_received")
	totalResponsesSent := expvar.NewInt("total_responses_sent")
	totalProcessingTimeMicroseconds := expvar.NewInt("total_processing_time_μs")
	// Declare a new expvar map to hold the count of responses for each HTTP status
	// code.
	totalResponsesSentByStatus := expvar.NewMap("total_responses_sent_by_status")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Increment the requests received count, like before.
		totalRequestsReceived.Add(1)
		// Call the httpsnoop.CaptureMetrics() function, passing in the next handler in
		// the chain along with the existing http.ResponseWriter and http.Request. This
		// returns the metrics struct that we saw above.
		metrics := httpsnoop.CaptureMetrics(next, w, r)
		// Increment the response sent count, like before.
		totalResponsesSent.Add(1)
		// Get the request processing time in microseconds from httpsnoop and increment
		// the cumulative processing time.
		totalProcessingTimeMicroseconds.Add(metrics.Duration.Microseconds())
		// Use the Add() method to increment the count for the given status code by 1.
		// Note that the expvar map is string-keyed, so we need to use the strconv.Itoa()
		// function to convert the status code (which is an integer) to a string.
		totalResponsesSentByStatus.Add(strconv.Itoa(metrics.Code), 1)
	})
	}

func CreateMiddlewareStack(xs ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for i := len(xs) - 1; i >= 0; i-- {
			x := xs[i]
			next = x(next)
		}
		return next
	}
}