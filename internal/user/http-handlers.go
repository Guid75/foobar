package user

import (
	"database/sql"
	"log"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

func NewUserHandler(w http.ResponseWriter, r *http.Request) {
	if err := UserForm().Render(r.Context(), w); err != nil {
		log.Println(err)
	}
}

func UsersHandler(w http.ResponseWriter, r *http.Request) {
	// ...
}

func CreateUserHandlerFactory(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		result, err := db.Exec(`
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

		newUser := User{
			ID:       int(id),
			Username: username,
			Role:     "user",
		}

		if err := UserRow(newUser).Render(r.Context(), w); err != nil {
			log.Println(err)
		}
	}
}

func DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	// ...
}
