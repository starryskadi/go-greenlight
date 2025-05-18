package data

import (
	"database/sql"
	"errors"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
	ErrDuplicateEmail = errors.New("duplicate email")
)

type Models struct {
	Movies       MovieInterface
	Genres       GenreModel
	MoviesGenres MoviesGenresModel
	Users        UserModel
	Tokens       TokenModel
	Permissions  PermissionModel
}

func NewModels(db *sql.DB) Models {
	return Models{
		Movies:       MovieModel{DB: db},
		Genres:       GenreModel{DB: db},
		MoviesGenres: MoviesGenresModel{DB: db},
		Users:        UserModel{DB: db},
		Tokens:       TokenModel{DB: db},
		Permissions:  PermissionModel{DB: db},
	}
}

func NewMockModel() Models {
	return Models{
		Movies: MockMovieModel{},
	}
}
