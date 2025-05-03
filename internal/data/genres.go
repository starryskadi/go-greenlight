package data

import (
	"database/sql"
)

type Genre struct {
	ID int 			`json:"-"`
	Title string 	`json:"title"`
}

type GenreModel struct {
	DB *sql.DB
}


func (g GenreModel) Insert(genre *Genre) error {
	stmt := "INSERT INTO genres (title) VALUES ($1) ON CONFLICT (title) DO UPDATE SET title = EXCLUDED.title RETURNING id;"

	row := g.DB.QueryRow(stmt, genre.Title)

	err := row.Scan(&genre.ID)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}

	return nil
}

func (g GenreModel) Update(genre Genre) error {
	// Won't Implement
	return nil
}

func (g GenreModel) Get(id int) (*Genre, error) {
	// Won't Implement
	return nil, nil
}

func (g GenreModel) Delete(id int) error {
	return nil
}