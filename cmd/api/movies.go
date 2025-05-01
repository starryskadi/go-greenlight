package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"kyawzayarwin.com/greenlight/internal/data"
	"kyawzayarwin.com/greenlight/internal/validator"
)

func (app *application) createMovieHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title string 		`json:"title"`
		Year int32			`json:"year"`
		Runtime data.Runtime`json:"runtime"`
		Genres []string 	`json:"genres"`
	}

	err := app.readJSON(w, r, &input)

	if err != nil {
		app.errorResponse(w, http.StatusInternalServerError, err.Error())
	}

	v := validator.New()

	movie := &data.Movie{
		Title: input.Title,
		Year: input.Year,
		Runtime: input.Runtime,
		Genres: input.Genres,
	}

	if data.ValidateMovie(v, movie); !v.Valid() {
		app.failedValidationResponse(w, v.Errors)
		return
	}

	err = app.models.Movies.Insert(movie)

	if err != nil {
		app.errorResponse(w,  http.StatusInternalServerError, err.Error())
	}

	for _, v := range movie.Genres {
		genre := data.Genre{
			Title: v,
		}

		err = app.models.Genres.Insert(&genre)

		if err != nil {
			app.errorResponse(w,  http.StatusInternalServerError, err.Error())
			return 
		}

		movieGenres := data.MoviesGenres{
			MovieID: movie.ID,
			GenreID: genre.ID,
		}

		err = app.models.MoviesGenres.AddMovieToGenre(movieGenres)

		if err != nil {
			app.errorResponse(w,  http.StatusInternalServerError, err.Error())
			return
		}
	}

	fmt.Fprintf(w, "%+v\n", input)
}

func (app *application) showMovieHandler(w http.ResponseWriter, r *http.Request) {
	idString := r.PathValue("id")

	id, err := strconv.Atoi(idString)

	if err != nil || id < 1 {
		http.NotFound(w, r)
		return
	}
	 
	movie := data.Movie{
		ID: id, 
		CreatedAt: time.Now(),
		Title: "Dummy Movie", 
		Year: 2025,
		Runtime: 300,
		Genres: []string{"Action"},
		Version: 1,
	}

	err = app.writeJSON(w, 200, envelope{ "movies": movie })

	if err != nil {
		app.serverErrorResponse(w, err)
	}
}

func (app *application) updateMovieHandler(w http.ResponseWriter, r *http.Request) {
	 w.Write([]byte("Update Movie Handler"))
}

func (app *application) deleteMovieHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Delete Movie Handler"))
}