package data

import (
	"database/sql"
	"errors"
)


var (
	ErrRecordNotFound = errors.New("record not found")
)

type Models struct {
	Movies MovieInterface
	Genres GenreModel
	MoviesGenres MoviesGenresModel
}

func NewModels(db *sql.DB) Models {
	return Models{
		Movies: MovieModel{DB: db},
		Genres: GenreModel{DB: db},
		MoviesGenres: MoviesGenresModel{ DB: db },
	}
}

func NewMockModel() Models {
	return Models{
		Movies: MockMovieModel{},
	}
}