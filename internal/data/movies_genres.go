package data

import (
	"database/sql"
)

type MoviesGenres struct {
	MovieID int 
	GenreID int
}


type MoviesGenresModel struct {
	DB *sql.DB
}

func (mg MoviesGenresModel) AddMovieToGenre(moviesGenres MoviesGenres) error {
	stmt := "INSERT INTO movies_genres (movie_id, genre_id) VALUES ($1, $2);"

	_, err := mg.DB.Exec(stmt, moviesGenres.MovieID, moviesGenres.GenreID)

	if err != nil {
		return err
	}

	return nil 
}   

func (mg MoviesGenresModel) DeleteMovieFromGenre(moviesGenres MoviesGenres) error {
	stmt := "DELETE FROM movies_genres WHERE movie_id = $1 AND genre_id = $2;"

	_, err := mg.DB.Exec(stmt, moviesGenres.MovieID, moviesGenres.GenreID)

	if err != nil {
		return err
	}

	return nil 
}   