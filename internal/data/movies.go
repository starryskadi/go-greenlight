package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
	"kyawzayarwin.com/greenlight/internal/validator"
)

type Movie struct {
	ID int				`json:"id"`
	CreatedAt time.Time	`json:"-"`
	Title string 		`json:"title"`
	Year int32			`json:"year"`
	Runtime Runtime 	`json:"runtime"`
	Genres []string 	`json:"genres"`
	Version int32		`json:"version"`
	// the visibility of individual struct fields in the JSON by using the omitempty and - struct tag directives.
}

func ValidateMovie(v *validator.Validator, movie *Movie) {
	v.Check(movie.Title != "", "title", "must be provided")
	v.Check(len(movie.Title) <= 500, "title", "must not be more than 500 bytes long")
	v.Check(movie.Year != 0, "year", "must be provided")
	v.Check(movie.Year >= 1888, "year", "must be greater than 1888")
	v.Check(movie.Year <= int32(time.Now().Year()), "year", "must not be in the future")
	v.Check(movie.Runtime != 0, "runtime", "must be provided")
	v.Check(movie.Runtime > 0, "runtime", "must be a positive integer")
	v.Check(movie.Genres != nil, "genres", "must be provided")
	v.Check(len(movie.Genres) >= 1, "genres", "must contain at least 1 genre")
	v.Check(len(movie.Genres) <= 5, "genres", "must not contain more than 5 genres")
	// values in the movie.Genres slice are unique.
	v.Check(validator.Unique(movie.Genres), "genres", "must not contain duplicate values")
}

type MovieModel struct {
	DB *sql.DB
}

type MovieInterface interface {
	Insert(movie *Movie) error
	Get(id int) (*Movie, error)
	Update(movie *Movie) error 
	Delete(id int) error
	GetAll(title string, genres []string, filters Filters) ([]*Movie, Metadata, error) 
}

func (m MovieModel) Insert(movie *Movie) error {
	stmt := `INSERT INTO movies (title, year, runtime) VALUES($1, $2, $3) RETURNING id, created_at, version;`

	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 3)
	defer cancel()

	return m.DB.QueryRowContext(ctx, stmt, movie.Title, movie.Year, movie.Runtime).Scan(&movie.ID, &movie.CreatedAt, &movie.Version)
}

func (m MovieModel) Get(id int) (*Movie, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	stmt := `SELECT m.id, m.title, m.year, m.runtime, m.version, ARRAY_AGG(g.title) as "genre_title" FROM public."movies" as m
		LEFT JOIN movies_genres as mg ON m.id = mg.movie_id
		LEFT JOIN genres as g ON mg.genre_id = g.id
		WHERE m.id = $1
		GROUP BY m.id,  m.title, m.year, m.runtime, m.version;`

	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 3)
	defer cancel()

	row := m.DB.QueryRowContext(ctx, stmt, id)  

	movie := &Movie{}

	var genreTitles []sql.NullString
	err := row.Scan(&movie.ID, &movie.Title, &movie.Year, &movie.Runtime, &movie.Version, pq.Array(&genreTitles))

	genres := []string{}
	for _, g := range genreTitles {
		if g.Valid {
			genres = append(genres, g.String)
		}
	}

	movie.Genres = genres

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}

	return movie, nil
}

func (m MovieModel) Update(movie *Movie) error {
	stmt := "UPDATE movies SET title = $2, year = $3, runtime = $4, version = version + 1 WHERE id = $1 AND version = $5 RETURNING version";

	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 3)
	defer cancel()

	row := m.DB.QueryRowContext(ctx, stmt, movie.ID, movie.Title, movie.Year, movie.Runtime, movie.Version);

	err := row.Scan(&movie.Version)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}

	return nil
}

func (m MovieModel) Delete(id int) error {
	stmt := "DELETE FROM movies WHERE id = $1;";

	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 3)
	defer cancel()

	result, err := m.DB.ExecContext(ctx, stmt, id)
	
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()

	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil 
}

func (m MovieModel) GetAll(title string, genres []string, filters Filters) ([]*Movie, Metadata, error) {
	stmt := fmt.Sprintf(`SELECT count(*) OVER(), m.id, m.title, m.year, m.runtime, m.version, ARRAY_AGG(g.title) as "genre_title" FROM public."movies" as m
		LEFT JOIN movies_genres as mg ON m.id = mg.movie_id
		LEFT JOIN genres as g ON mg.genre_id = g.id
		WHERE (to_tsvector('simple', m.title) @@ plainto_tsquery('simple', $1) OR $1 = '') 
		GROUP BY m.id,  m.title, m.year, m.runtime, m.version
		HAVING ($2 <@ ARRAY_AGG(g.title) OR $2= '{}')
		ORDER BY m.%s %s
		LIMIT $3
		OFFSET $4;`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 3)
	defer cancel()

	args := []any{title, pq.Array(genres), filters.limit(), filters.offset()}

	row, err := m.DB.QueryContext(ctx, stmt,args...)  

	if err != nil {
		return nil, Metadata{}, err
	}

	movies := []*Movie{}
	var totalRecords int

	for row.Next() {
		var movie Movie

		var genreTitles []sql.NullString

		err := row.Scan(&totalRecords, &movie.ID, &movie.Title, &movie.Year, &movie.Runtime, &movie.Version, pq.Array(&genreTitles))

		genres := []string{}
		for _, g := range genreTitles {
			if g.Valid {
				genres = append(genres, g.String)
			}
		}

		movie.Genres = genres

		if err != nil {
			return nil, Metadata{}, err
		}

		movies = append(movies, &movie)
	}

	metadata := calculateMetadata(totalRecords,filters.Page, filters.PageSize)

	if err = row.Err(); err != nil {
		return nil, Metadata{}, err
	}

	return movies, metadata, nil 
}

type MockMovieModel struct{}

func (m MockMovieModel) Insert(movie *Movie) error {
	// Mock the action...
	return nil 
}
func (m MockMovieModel) Get(id int) (*Movie, error) {
	return nil, nil
}
func (m MockMovieModel) Update(movie *Movie) error {
	return nil
}

func (m MockMovieModel) Delete(id int) error {
	return nil 
}

func (m MockMovieModel) GetAll(title string, genres []string, filters Filters) ([]*Movie, Metadata, error) {
	return nil, Metadata{}, nil
}