package data

import (
	"context"
	"crypto/sha256"
	"errors"
	"time"

	"database/sql"

	"golang.org/x/crypto/bcrypt"
	"kyawzayarwin.com/greenlight/internal/validator"
)

var AnonymousUser = &User{}

type User struct {
	ID			int64			`json:"id"`
	CreatedAt	time.Time		`json:"created_at"`
	Name 		string			`json:"name"`
	Email		string 			`json:"email"`
	Password	password		`json:"-"`
	Activated	bool			`json:"activated"`
	Version 	int				`json:"version"`
}

func (u *User) IsAnonymous() bool {
	return u == AnonymousUser
}

type password struct {
	plaintext *string
	hash []byte
}


func (p *password) Set(plaintextPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), 12)

	if err != nil {
		return nil 
	}

	p.plaintext = &plaintextPassword
	p.hash = hash

	return nil 
}

func (p *password) Matches(plaintextPassword string) (bool, error)  {
	err := bcrypt.CompareHashAndPassword(p.hash, []byte(plaintextPassword))
	if err != nil {
		switch {
			case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
			default:
			return false, err
		}
	}
	return true, nil
}

func ValidateEmail(v *validator.Validator, email string) {
	v.Check(email != "", "email", "must be provided")
	v.Check(validator.Matches(email, validator.EmailRX), "email", "must be a valid email address")
}

func ValidatePasswordPlaintext(v *validator.Validator, password string) {
	v.Check(password != "", "password", "must be provided")
	v.Check(len(password) >= 8, "password", "must be at least 8 bytes long")
	v.Check(len(password) <= 72, "password", "must not be more than 72 bytes long")
}

func ValidateUser(v *validator.Validator, user *User) {
	v.Check(user.Name != "", "name", "must be provided")
	v.Check(len([]rune(user.Name)) <= 300, "name", "must not be more than 300 letters long")

	ValidateEmail(v, user.Email)

	if user.Password.plaintext != nil {
		ValidatePasswordPlaintext(v, *user.Password.plaintext)
	}

	if user.Password.hash == nil {
		panic("missing password hash for user")
	}
}  

type UserModel struct {
	DB *sql.DB
}

func (u *UserModel) Insert(user *User) error {
	stmt := `
		INSERT INTO users (name, email, password_hash, activated)
		VALUES ($1, $2, $3, $4)	
		RETURNING id, created_at, version;`

	args := []any{user.Name, user.Email, user.Password.hash, user.Activated}

	ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
	defer cancel() 

	row := u.DB.QueryRowContext(ctx, stmt, args...)

	err := row.Scan(&user.ID, &user.CreatedAt, &user.Version)

	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicateEmail
		default:
			return err
		}
	}

	return nil
}

func (u *UserModel) GetByEmail(email string) (*User, error) {
	stmt := `
		SELECT id, name, email, password_hash, activated, version
		FROM users
		WHERE email = $1
	`

	ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
	defer cancel()

	var user User 

	row := u.DB.QueryRowContext(ctx, stmt,email)

	err := row.Scan(&user.ID, &user.Name, &user.Email, &user.Password.hash, &user.Activated, &user.Version) 

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, err 
		default:
			return nil, err 
		}
	}

	return &user, nil 
}

func (u *UserModel) Update(user *User) error {
	stmt := `
		UPDATE users 
		SET name = $1, email = $2, password_hash = $3, activated = $4, version = version + 1
		WHERE id = $5 and version = $6
		RETURNING version
	`

	ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
	defer cancel() 

	row := u.DB.QueryRowContext(ctx, stmt, user.Name, user.Email, user.Password.hash, user.Activated, user.ID, user.Version)

	err := row.Scan(&user.Version)

	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicateEmail
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}

	return nil 
}

func (u *UserModel) GetFromToken(tokenScope string, tokenPlainText string)  (*User, error) {
	tokenHash := sha256.Sum256([]byte(tokenPlainText))

	stmt := `SELECT users.id, users.created_at, users.name, users.email, users.password_hash, users.activated, users.version
		FROM users
		INNER JOIN tokens
		ON users.id = tokens.user_id
		WHERE tokens.hash = $1
		AND tokens.scope = $2
		AND tokens.expiry > $3`

	args := []any{tokenHash[:], tokenScope, time.Now()}

	var user User

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := u.DB.QueryRowContext(ctx, stmt, args...).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.Name,
		&user.Email,
		&user.Password.hash,
		&user.Activated, 
		&user.Version,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}

	}

	return &user, nil
}