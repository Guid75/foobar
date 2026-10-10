package main

import (
	"net/http"

	"github.com/Guid75/foobar/internal/user"
)

func (app *App) registerUserRoutes(mux *http.ServeMux) {
	mux.Handle(
		"GET /users/new",
		app.requireSuperAdmin(
			http.HandlerFunc(user.NewUserHandler),
		),
	)

	mux.Handle(
		"GET /users",
		app.requireSuperAdmin(
			http.HandlerFunc(user.UsersHandlerFactory(app.database)),
		),
	)

	mux.Handle(
		"POST /users",
		app.requireSuperAdmin(
			http.HandlerFunc(user.CreateUserHandlerFactory(app.database)),
		),
	)

	mux.Handle(
		"POST /users/{id}/delete",
		app.requireSuperAdmin(
			http.HandlerFunc(user.DeleteUserHandler),
		),
	)
}
