package user

import (
	"database/sql"
)

func FindByID(db *sql.DB, id int) (*User, error) {
	var user User

	err := db.QueryRow(`
        SELECT id, username, role
        FROM users
        WHERE id = ?
    `, id).Scan(
		&user.ID,
		&user.Username,
		&user.Role,
	)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func List(db *sql.DB) ([]User, error) {
	rows, err := db.Query(`
        SELECT id, username, role
        FROM users
        ORDER BY username
    `)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User

	for rows.Next() {
		var user User

		if err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Role,
		); err != nil {
			return nil, err
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}
