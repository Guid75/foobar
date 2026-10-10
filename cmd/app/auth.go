package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/Guid75/foobar/internal/user"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

func authenticate(database *sql.DB, username, password string) (int, error) {
	var (
		userID       int
		passwordHash string
	)

	err := database.QueryRow(`
        SELECT id, password_hash
        FROM users
        WHERE username = ?
    `, username).Scan(&userID, &passwordHash)

	if err == sql.ErrNoRows {
		return 0, ErrInvalidCredentials
	}

	if err != nil {
		return 0, err
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(passwordHash),
		[]byte(password)); err != nil {
		return 0, ErrInvalidCredentials
	}

	return userID, nil
}

type Session struct {
	UserID int
}

var (
	sessions   = make(map[string]Session)
	sessionsMu sync.RWMutex
)

func generateSessionToken() (string, error) {
	data := make([]byte, 32)

	if _, err := rand.Read(data); err != nil {
		return "", err
	}

	return hex.EncodeToString(data), nil
}

func getSession(r *http.Request) (Session, bool) {
	cookie, err := r.Cookie("session")
	if err != nil {
		return Session{}, false
	}

	sessionsMu.RLock()
	session, ok := sessions[cookie.Value]
	sessionsMu.RUnlock()

	return session, ok
}

type contextKey int

const userContextKey contextKey = iota

func (app *App) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, ok := getSession(r)
		if !ok {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		user, err := user.FindByID(app.database, session.UserID)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func (app *App) requireRole(role string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := getCurrentUserFromContext(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if user.Role != role {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (app *App) requireSuperAdmin(next http.Handler) http.Handler {
	return app.requireAuth(
		app.requireRole("super_admin", next),
	)
}

func (app *App) loginPageHandler(w http.ResponseWriter, r *http.Request) {
	_, ok := getSession(r)

	if ok {
		http.Redirect(w, r, "/main", http.StatusSeeOther)
		return
	}

	if err := LoginPage().Render(r.Context(), w); err != nil {
		log.Println(err)
	}
}

func (app *App) loginHandler(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	userID, err := authenticate(app.database, username, password)

	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}

		log.Println(err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	token, err := generateSessionToken()
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	sessionsMu.Lock()
	sessions[token] = Session{
		UserID: userID,
	}
	sessionsMu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/main", http.StatusSeeOther)
}

func (app *App) logoutHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("logout")
	cookie, err := r.Cookie("session")
	if err == nil {
		sessionsMu.Lock()
		delete(sessions, cookie.Value)
		sessionsMu.Unlock()
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/login")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *App) currentUser(r *http.Request) (*user.User, bool) {
	session, ok := getSession(r)
	if !ok {
		return nil, false
	}

	user, err := user.FindByID(app.database, session.UserID)
	if err != nil {
		return nil, false
	}

	return user, true
}

func getCurrentUserFromContext(r *http.Request) (*user.User, bool) {
	user, ok := r.Context().Value(userContextKey).(*user.User)
	return user, ok
}
