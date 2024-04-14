package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"database/sql"

	"github.com/gorilla/mux"
)

type user struct {
	Id          int     `json:"id"`
	Name        string  `json:"name"`
	Lastname    string  `json:"lastname"`
	Avatar      *string `json:"avatar"`
	Description *string `json:"description"`
	News        int     `json:"newsCount"`
}

func getUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	result := db.QueryRow("select name, lastname, avatar, description, (select count(*) FROM news where user_id=users.id and deleted=false) as comms from users where id=$1 and deleted=false", id)

	u := user{}
	err := result.Scan(&u.Name, &u.Lastname, &u.Avatar, &u.Description, &u.News)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	jsonUser, err := json.Marshal(u)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, string(jsonUser))
}

func partUpdateUser(w http.ResponseWriter, r *http.Request) {
	rl, err := auth(w, r)
	if err != nil {
		return
	}

	var u user
	err = json.NewDecoder(r.Body).Decode(&u)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	rows, err := db.Query("update users set description=$1 where id=$2 RETURNING description", u.Description, rl.id)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer rows.Close()
	if rows.Next() {
		err = rows.Scan(&u.Description)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

	} else {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	jsonUser, err := json.Marshal(u)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, string(jsonUser))
}

func deleteUser(w http.ResponseWriter, r *http.Request) {
	rl, err := auth(w, r)
	if err != nil {
		return
	}

	_, err = db.Query("update users set deleted=true where id=$1", rl.id)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	Logout(w, r)
}

func getUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("select id, name, lastname, avatar, description from users where deleted=false order by id asc")
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer rows.Close()
	usersCollection := []user{}

	for rows.Next() {
		u := user{}
		err := rows.Scan(&u.Id, &u.Name, &u.Lastname, &u.Avatar, &u.Description)
		if err != nil {
			if err == sql.ErrNoRows {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		usersCollection = append(usersCollection, u)
	}

	jsonUsers, err := json.Marshal(usersCollection)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, string(jsonUsers))
}

func updateUserAvatar(w http.ResponseWriter, r *http.Request) {
	rl, err := auth(w, r)
	if err != nil {
		return
	}

	date := time.Now().UTC()

	// Request validation
	file, _, err := r.FormFile("avatar")
	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	photoId, err := loadImage(file, rl, date, true)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	u, err := setUserAvatar(rl, &photoId)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	jsonUser, err := json.Marshal(u)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, string(jsonUser))
}

func deleteUserAvatar(w http.ResponseWriter, r *http.Request) {
	rl, err := auth(w, r)
	if err != nil {
		return
	}

	u, err := setUserAvatar(rl, nil)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	jsonUser, err := json.Marshal(u)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, string(jsonUser))
}

func setUserAvatar(rl role, avatar *string) (user, error) {
	rows, err := db.Query("update users set avatar=$1 where id=$2 RETURNING id, name, lastname, avatar, (select avatar from users where id=$2) as last_avatar;", avatar, rl.id)
	if err != nil {
		return user{}, err
	}
	defer rows.Close()

	if rows.Next() {
		var lastAvatar *string = nil
		u := user{}
		err := rows.Scan(&u.Id, &u.Name, &u.Lastname, &u.Avatar, &lastAvatar)
		if err != nil {
			return user{}, err
		}

		// Clean last avatar
		if lastAvatar != nil {
			go cleanPhotos([]string{*lastAvatar}, true)
		}

		return u, nil
	} else {
		return user{}, fmt.Errorf("no returning")
	}
}
