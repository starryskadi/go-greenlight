package data

import (
	"context"
	"database/sql"
	"time"

	"github.com/lib/pq"
)

const (
	PermissionMovieRead  = "movies:read"
	PermissionMovieWrite = "movies:write"
)

type Permissions []string

func (p Permissions) Include(code string) bool {
	for i := range p {
		if code == p[i] {
			return true
		}
	}

	return false
}

type PermissionModel struct {
	DB *sql.DB
}

func (pm PermissionModel) GetAllForUser(userID int64) (Permissions, error) {
	stmt := `
		SELECT permissions.code
		FROM permissions
		INNER JOIN users_permissions ON users_permissions.permission_id = permissions.id
		INNER JOIN users ON users_permissions.user_id = users.id
		WHERE users.id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := pm.DB.QueryContext(ctx, stmt, userID)

	permissions := Permissions{}

	for rows.Next() {
		var permission string
		err := rows.Scan(&permission)

		if err != nil {
			return nil, err
		}

		permissions = append(permissions, permission)
	}

	if rows.Err(); err != nil {
		return nil, err
	}

	return permissions, nil
}

func (pm PermissionModel) AddForUser(userID int64, codes ...string) error {
	stmt := `INSERT INTO users_permissions SELECT $1, permissions.id FROM permissions WHERE permissions.code = ANY($2)`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []any{userID, pq.Array(codes)}

	_, err := pm.DB.ExecContext(ctx, stmt, args...)

	return err
}
