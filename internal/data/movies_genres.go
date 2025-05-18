package data

import (
	"database/sql"
	"fmt"
	"strings"
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

func (mg MoviesGenresModel) BulkUpdateMoviesFromGenre(movieID int, moviesGenres []MoviesGenres) error {
	if len(moviesGenres) < 1 {
		stmt := `DELETE FROM movies_genres WHERE movie_id = $1;`
		_, err := mg.DB.Exec(stmt, movieID)

		if err != nil {
			return err
		}

		return nil
	}

	s := []string{}

	for i := range moviesGenres {
		s = append(s, fmt.Sprintf("($%d::int, $%d::int)", i*2+1, i*2+2))
	}

	sClause := strings.Join(s, ",")

	stmt := fmt.Sprintf(` 
	WITH synced (movie_id, genre_id) AS (
		VALUES %s
	),
	upsert AS (
		INSERT INTO movies_genres (movie_id, genre_id)
		SELECT movie_id, genre_id FROM synced
		ON CONFLICT (movie_id, genre_id) DO UPDATE 
		SET movie_id = EXCLUDED.movie_id, genre_id = EXCLUDED.genre_id
	)
	DELETE FROM movies_genres
	WHERE (movie_id, genre_id) NOT IN (SELECT movie_id, genre_id FROM synced);`, sClause)

	val := []any{}

	for _, v := range moviesGenres {
		val = append(val, v.MovieID, v.GenreID)
	}

	_, err := mg.DB.Exec(stmt, val...)

	if err != nil {
		return err
	}

	return nil
}
