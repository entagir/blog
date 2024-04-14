package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"database/sql"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type session struct {
	Id     int       `json:"id"`
	Login  string    `json:"login"`
	Expiry time.Time `json:"expiry"`
}

func (s session) isExpired() bool {
	return s.Expiry.Before(time.Now())
}

type role struct {
	id int
}

func auth(w http.ResponseWriter, r *http.Request) (role, error) {
	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			w.WriteHeader(http.StatusUnauthorized)
			return role{}, fmt.Errorf("no cookie")
		}
		w.WriteHeader(http.StatusBadRequest)
		return role{}, fmt.Errorf("error")
	}
	sessionToken := c.Value

	val, err := rdb.Get(ctx, sessionToken).Result()
	if err != nil || val == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return role{}, fmt.Errorf("not exist session")
	}

	var userSession session
	err = json.Unmarshal([]byte(val), &userSession)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return role{}, fmt.Errorf("not exist session")
	}
	if userSession.isExpired() {
		rdb.Del(ctx, sessionToken)
		w.WriteHeader(http.StatusUnauthorized)
		return role{}, fmt.Errorf("session expired")
	}

	return role{userSession.Id}, nil
}

func Signup(w http.ResponseWriter, r *http.Request) {
	login := r.FormValue("login")
	password := r.FormValue("password")
	name := r.FormValue("name")
	lastname := r.FormValue("lastname")

	if login == "" || password == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	rows, err := db.Query("select * from users where login=$1 and deleted=false", login)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	if rows.Next() {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "User \""+login+"\" already exist")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 8)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, err = db.Query("insert into users (login, password, name, lastname) values ($1, $2, $3, $4)", login, string(hashedPassword), name, lastname); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func Signin(w http.ResponseWriter, r *http.Request) {
	var id int
	login := r.FormValue("login")
	password := r.FormValue("password")

	result := db.QueryRow("select id, password from users where login=$1 and deleted=false", login)

	var storedPassword string
	err := result.Scan(&id, &storedPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, "User \""+login+"\" not exist")
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err = bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(password)); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Err login or password")
		return
	}

	sessionToken := uuid.NewString()
	expiresAt := time.Now().Add(24 * 180 * time.Hour)

	// sessions[sessionToken] = session{
	// 	Id:     id,
	// 	Login:  login,
	// 	Expiry: expiresAt,
	// }

	jsonSession, err := json.Marshal(session{
		Id:     id,
		Login:  login,
		Expiry: expiresAt,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = rdb.Set(ctx, sessionToken, jsonSession, time.Duration(24*180*time.Hour)).Err()
	if err != nil {
		fmt.Println(err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   sessionToken,
		Expires: expiresAt,
		Path:    "/",
	})
	http.SetCookie(w, &http.Cookie{
		Name:  "id",
		Value: fmt.Sprint(id),
		Path:  "/",
	})

	http.Redirect(w, r, "/id"+fmt.Sprint(id), http.StatusSeeOther)
}

func Logout(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			http.Redirect(w, r, "/sign", http.StatusSeeOther)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	sessionToken := c.Value

	rdb.Del(ctx, sessionToken)

	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   "",
		Expires: time.Now(),
		Path:    "/",
	})
	http.SetCookie(w, &http.Cookie{
		Name:    "id",
		Value:   "",
		Expires: time.Now(),
		Path:    "/",
	})

	http.Redirect(w, r, "/sign", http.StatusSeeOther)
}

func Ping(w http.ResponseWriter, r *http.Request) {
	rl, err := auth(w, r)
	if err != nil {
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, fmt.Sprint("{\"id\":", rl.id, "}"))
}
