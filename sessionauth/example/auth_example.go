// Auth example is an example application which requires a login
// to view a private link. The username is "testuser" and the password
// is "password". This will require GORP and an SQLite3 database.
package main

import (
	"database/sql"
	"github.com/coopernurse/gorp"
	_ "github.com/mattn/go-sqlite3"
	"github.com/treetopllc/martini"
	"github.com/treetopllc/martini-contrib/binding"
	"github.com/treetopllc/martini-contrib/render"
	"github.com/treetopllc/martini-contrib/sessionauth"
	"github.com/treetopllc/martini-contrib/sessions"
	"log"
	"net/http"
	"os"
)

var dbmap *gorp.DbMap

func initDb() *gorp.DbMap {
	// Delete our SQLite database if it already exists so we have a clean start
	_, err := os.Open("martini-sessionauth.bin")
	if err == nil {
		os.Remove("martini-sessionauth.bin")
	}

	db, err := sql.Open("sqlite3", "martini-sessionauth.bin")
	if err != nil {
		log.Fatalln("Fail to create database", err)
	}

	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}
	dbmap.AddTableWithName(MyUserModel{}, "users").SetKeys(true, "Id")
	err = dbmap.CreateTablesIfNotExists()
	if err != nil {
		log.Fatalln("Could not build tables", err)
	}

	user := MyUserModel{1, "testuser", "password", false}
	err = dbmap.Insert(&user)
	if err != nil {
		log.Fatalln("Could not insert test user", err)
	}
	return dbmap
}

func main() {
	store := sessions.NewCookieStore([]byte("secret123"))
	dbmap = initDb()

	m := martini.Classic()
	m.Use(render.Renderer())
	m.Use(sessions.Sessions("my_session", store))
	m.Use(sessionauth.SessionUser(GenerateAnonymousUser))

	m.Get("/", func(r render.Render) {
		r.HTML(200, "index", nil)
	})

	m.Get("/login", func(r render.Render) {
		r.HTML(200, "login", nil)
	})

	m.Post("/login", binding.Bind(MyUserModel{}), func(session sessions.Session, postedUser MyUserModel, r render.Render, req *http.Request) {
		// You should verify credentials against a database or some other mechanism at this point.
		// Then you can authenticate this session.
		user := MyUserModel{}
		err := dbmap.SelectOne(&user, "SELECT * FROM users WHERE username = $1 and password = $2", postedUser.Username, postedUser.Password)
		if err != nil {
			r.Redirect("/login")
			return
		} else {
			err := sessionauth.AuthenticateSession(session, &user)
			if err != nil {
				r.JSON(500, err)
			}

			params := req.URL.Query()
			redirect := params.Get("next")
			r.Redirect(redirect)
			return
		}
	})

	m.Get("/private", sessionauth.LoginRequired, func(r render.Render, user sessionauth.User) {
		r.HTML(200, "private", user.(*MyUserModel))
	})

	m.Get("/logout", sessionauth.LoginRequired, func(session sessions.Session, user sessionauth.User, r render.Render) {
		sessionauth.Logout(session, user)
		r.Redirect("/")
	})

	m.Run()
}
