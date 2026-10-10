package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"syscall"

	"github.com/Guid75/foobar/internal/db"
	"github.com/Guid75/foobar/internal/user"
	"github.com/charmbracelet/x/term"
	"golang.org/x/crypto/bcrypt"
)

type App struct {
	database *sql.DB
}

func newApp(database *sql.DB) (*App, error) {
	return &App{
		database: database,
	}, nil
}

func main() {
	database, err := db.Open("app.db")
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	switch {
	case len(os.Args) == 1:
		app, err := newApp(database)
		if err != nil {
			log.Fatal(err)
		}
		startServer(app)

	case os.Args[1] == "create-admin":
		createAdmin(database)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func checkAdmin(database *sql.DB) {
	var count int
	err := database.QueryRow(`
        SELECT COUNT(*)
        FROM users
        WHERE role = 'super_admin'
    `).Scan(&count)

	if err != nil {
		log.Fatal(err)
	}

	if count == 0 {
		log.Println("WARNING: no super-admin account exists.")
	}
}

func createAdmin(database *sql.DB) {
	var count int

	err := database.QueryRow(`
        SELECT COUNT(*)
        FROM users
        WHERE role = 'super_admin'
    `).Scan(&count)

	if err != nil {
		log.Fatal(err)
	}

	if count > 0 {
		log.Fatal("A super-admin already exists.")
	}

	username := "admin"

	fmt.Print("Enter the super admin password of your choice: ")
	password, err := readPassword()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print("Confirm password: ")
	confirmation, err := readPassword()
	if err != nil {
		log.Fatal(err)
	}

	if password != confirmation {
		log.Fatal("Passwords do not match")
	}

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = database.Exec(`
		INSERT INTO users (username, password_hash, role)
		VALUES (?, ?, ?)
	`, username, string(passwordHash), "super_admin")

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Administrator created.")
}

func readPassword() (string, error) {
	password, err := term.ReadPassword(uintptr(syscall.Stdin))
	fmt.Println()

	return string(password), err
}

func startServer(app *App) {
	checkAdmin(app.database)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", app.loginPageHandler)

	mux.HandleFunc("POST /auth/login", app.loginHandler)

	mux.HandleFunc("POST /auth/logout", app.logoutHandler)

	mux.Handle("GET /dashboard", app.requireAuth(http.HandlerFunc(app.dashboardHandler)))

	app.registerUserRoutes(mux)

	log.Println("Server listening on http://localhost:8080")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

func (app *App) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := getCurrentUserFromContext(r)
	if !ok {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := DashboardData{
		User: *currentUser,
	}

	if currentUser.Role == "super_admin" {
		users, err := user.List(app.database)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		data.Users = users
	}

	if err := DashboardPage(data.User, data.Users).Render(r.Context(), w); err != nil {
		log.Println(err)
	}
}

type DashboardData struct {
	User  user.User
	Users []user.User
}
