package data

import (
	"database/sql"
	"errors"
	"time"

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
}

func (m MovieModel) Insert(movie *Movie) error {
	stmt := `INSERT INTO movies (title, year, runtime) VALUES($1, $2, $3) RETURNING id, created_at, version;`

	return m.DB.QueryRow(stmt, movie.Title, movie.Year, movie.Runtime).Scan(&movie.ID, &movie.CreatedAt, &movie.Version)
}

func (m MovieModel) Get(id int) (*Movie, error) {
	stmt := `SELECT m.id, m.title, m.year, m.runtime, g.id as "genre_id" , g.title AS "genre_title" FROM public."movies" as m
		LEFT JOIN movies_genres as mg ON m.id = mg.movie_id
		JOIN genres as g ON mg.genre_id = g.id
		WHERE m.id = $1;`

	rows, err := m.DB.Query(stmt, id)  

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	} 

	defer rows.Close()

	movie := &Movie{}

	genreMap := map[int]bool{}

	found := false

	for rows.Next() {
		found = true
		var (
			ID int
			title string 
			year int32
			runtime int 
			genreId sql.NullInt32
			genreTitle sql.NullString  
		)


		err = rows.Scan(&ID, &title, &year, &runtime, &genreId, &genreTitle)

		if err != nil {
			return nil, err
		} 

		if movie.ID == 0 {
			movie.ID = ID 
			movie.Title = title 
			movie.Year = year 
			movie.Runtime = Runtime(runtime)  
		}

		if genreId.Valid && genreTitle.Valid {
			if _, exists := genreMap[int(genreId.Int32)]; !exists {
				movie.Genres = append(movie.Genres, genreTitle.String)
				genreMap[int(genreId.Int32)] = true 
			}
		}
	}

	if !found {
		return nil, ErrRecordNotFound
	}

	return movie, nil
}

func (m MovieModel) Update(movie *Movie) error {
	stmt := "UPDATE movies SET title = $2, year = $3, runtime = $4 WHERE id = $1";

	_, err := m.DB.Exec(stmt, movie.ID, movie.Title, movie.Year, movie.Runtime);

	if err != nil {
		return err 
	}

	return nil
}

func (m MovieModel) Delete(id int) error {
	return nil 
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