package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

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
		app.errorResponse(w, r, http.StatusInternalServerError, err.Error())
	}

	v := validator.New()

	movie := &data.Movie{
		Title: input.Title,
		Year: input.Year,
		Runtime: input.Runtime,
		Genres: input.Genres,
	}

	if data.ValidateMovie(v, movie); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Movies.Insert(movie)

	if err != nil {
		app.errorResponse(w, r, http.StatusInternalServerError, err.Error())
	}

	for _, v := range movie.Genres {
		genre := data.Genre{
			Title: v,
		}

		err = app.models.Genres.Insert(&genre)

		if err != nil {
			app.errorResponse(w, r, http.StatusInternalServerError, err.Error())
			return 
		}

		movieGenres := data.MoviesGenres{
			MovieID: movie.ID,
			GenreID: genre.ID,
		}

		err = app.models.MoviesGenres.AddMovieToGenre(movieGenres)

		if err != nil {
			app.errorResponse(w, r, http.StatusInternalServerError, err.Error())
			return
		}
	}

	fmt.Fprintf(w, "%+v\n", input)
}

func (app *application) listMoviesHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title string 
		Genres []string 
		data.Filters
	}
	
	qs := r.URL.Query()

	v := validator.New();

	input.Title = app.readString(qs, "title", "")
	input.Genres = app.readCSV(qs, "genres", []string{})
	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 10, v)
	input.Filters.Sort = app.readString(qs, "sort", "id")
	input.Filters.SortSafelist = []string{"id", "title", "year", "runtime", "-id", "-title", "-year", "-runtime"}

	data.ValidateFilter(v, input.Filters)

	if !v.Valid() {
		app.failedValidationResponse(w,r,  v.Errors)
		return 
	}

	fmt.Printf("%+v\n", input)

	movies, metadata, err := app.models.Movies.GetAll(input.Title, input.Genres, input.Filters)

	if err != nil {
		app.serverErrorResponse(w,r,  err)
	}

	app.writeJSON(w, http.StatusOK, envelope{
		"movies": movies,
		"metadata": metadata,
	})
}

func (app *application) showMovieHandler(w http.ResponseWriter, r *http.Request) {
	idString := r.PathValue("id")

	id, err := strconv.Atoi(idString)

	if err != nil || id < 1 {
		app.notFoundResponse(w, r)
		return
	}
	 
	movie, err := app.models.Movies.Get(id)

	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w,r)
		default:
			app.serverErrorResponse(w,r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{ "movies": movie })

	if err != nil {
		app.serverErrorResponse(w,r, err)
	}
}

func (app *application) updateMovieHandler(w http.ResponseWriter, r *http.Request) {
	idString := r.PathValue("id")

	id, err := strconv.Atoi(idString)

	if err != nil || id < 1 {
		app.notFoundResponse(w,r)
		return 
	}

	movie, err := app.models.Movies.Get(id)
	if err != nil {
		switch {
			case errors.Is(err, data.ErrRecordNotFound):
				app.notFoundResponse(w,r)
			default:
				app.serverErrorResponse(w, r, err)
		}
		return 
	}
	

	var input struct {
		Title *string `json:"title"`
		Year *int32 `json:"year"`
		Runtime *data.Runtime `json:"runtime"`
		Genres []string `json:"genres"`
	} 

	err = app.readJSON(w, r, &input)

	if err != nil {
		app.badRequestResponse(w,r, err)
		return
	}

	if input.Title != nil {
		movie.Title = *input.Title
	}

	if input.Year != nil {
		movie.Year= *input.Year
	}

	if input.Runtime != nil {
		movie.Runtime= *input.Runtime
	}

	if input.Genres != nil {
		movie.Genres= input.Genres
	}

	v := validator.New()

	if data.ValidateMovie(v, movie);  !v.Valid() {
		app.failedValidationResponse(w,r, v.Errors)
		return
	}

	err = app.models.Movies.Update(movie)

	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w,r)
		default:
			app.serverErrorResponse(w,r, err) 
		}
		
		return 
	}

	moviesGenres := []data.MoviesGenres{}

	for _, v := range movie.Genres {
		genre := &data.Genre{ Title: v }
		
		err := app.models.Genres.Insert(genre)

		if err != nil {
			app.serverErrorResponse(w, r,err) 
			return
		}

		moviesGenres = append(moviesGenres, 
			data.MoviesGenres{
				MovieID: movie.ID,
				GenreID: genre.ID,
			},
		)
	}

	err  = app.models.MoviesGenres.BulkUpdateMoviesFromGenre(movie.ID, moviesGenres)

	if err != nil {
		app.serverErrorResponse(w, r, err) 
		return 
	}

	err = app.writeJSON(w, http.StatusOK, envelope{ "movies": movie })

	if err != nil {
		app.serverErrorResponse(w, r, err) 
		return 
	}
}

func (app *application) deleteMovieHandler(w http.ResponseWriter, r *http.Request) {
	idString := r.PathValue("id")

	id, err := strconv.Atoi(idString)

	if err != nil || id < 1 {
		app.notFoundResponse(w, r)
		return 
	}

	err = app.models.Movies.Delete(id)

	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err) 
		}
		return 
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"message": "movie successfully deleted"})

	if err != nil {
		app.serverErrorResponse(w,r, err) 
		return 
	}
}