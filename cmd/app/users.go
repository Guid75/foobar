package main

import (
	"context"
	"log"
	"net/http"

	"github.com/Guid75/foobar/internal/user"
	"golang.org/x/crypto/bcrypt"
)

func (app *App) registerUserRoutes(mux *http.ServeMux) {
	mux.Handle(
		"GET /users/new",
		app.requireSuperAdmin(
			http.HandlerFunc(app.newUserHandler),
		),
	)

	mux.Handle(
		"GET /users",
		app.requireSuperAdmin(
			http.HandlerFunc(app.usersHandler),
		),
	)

	mux.Handle(
		"POST /users",
		app.requireSuperAdmin(
			http.HandlerFunc(app.createUserHandler),
		),
	)

	mux.Handle(
		"POST /users/{id}/delete",
		app.requireSuperAdmin(
			http.HandlerFunc(app.deleteUserHandler),
		),
	)
}

func (app *App) newUserHandler(w http.ResponseWriter, r *http.Request) {
	if err := user.UserForm().Render(r.Context(), w); err != nil {
		log.Println(err)
	}
}

func (app *App) usersHandler(w http.ResponseWriter, r *http.Request) {
	// ...
}

func (app *App) createUserHandler(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	result, err := app.database.Exec(`
        INSERT INTO users (username, password_hash, role)
        VALUES (?, ?, ?)
    `, username, string(passwordHash), "user")

	if err != nil {
		// On traitera notamment le username déjà existant ici.
		http.Error(w, "Unable to create user", http.StatusBadRequest)
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	newUser := user.User{
		ID:       int(id),
		Username: username,
		Role:     "user",
	}

	if err := user.UserRow(newUser).Render(context.Background(), w); err != nil {
		log.Println(err)
	}
}

func (app *App) deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	// ...
}

func (app *App) findUserByID(id int) (*user.User, error) {
	var user user.User

	err := app.database.QueryRow(`
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

func (app *App) listUsers() ([]user.User, error) {
	rows, err := app.database.Query(`
        SELECT id, username, role
        FROM users
        ORDER BY username
    `)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []user.User

	for rows.Next() {
		var user user.User

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
