package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"database/sql"

	"github.com/gorilla/mux"
)

type comment struct {
	NewsId int    `json:"newsId"`
	Text   string `json:"text"`
}

type commentItem struct {
	Id           int       `json:"id"`
	UserId       int       `json:"userId"`
	UserName     string    `json:"userName"`
	UserLastName string    `json:"userLastName"`
	UserAvatar   *string   `json:"userAvatar"`
	Text         string    `json:"text"`
	Date         time.Time `json:"date"`
}

func addComment(w http.ResponseWriter, r *http.Request) {
	rl, err := auth(w, r)
	if err != nil {
		return
	}

	var c comment
	err = json.NewDecoder(r.Body).Decode(&c)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check allow comments for news

	date := time.Now().UTC()
	rows, err := db.Query("insert into comments (user_id, news_id, text, date) values ($1, $2, $3, $4) RETURNING id, user_id, text, date", rl.id, c.NewsId, c.Text, date)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	defer rows.Close()
	commentsCollection := []commentItem{}

	if rows.Next() {
		n := commentItem{}
		err = rows.Scan(&n.Id, &n.UserId, &n.Text, &n.Date)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		commentsCollection = append(commentsCollection, n)

		jsonComments, err := json.Marshal(commentsCollection)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		fmt.Fprint(w, string(jsonComments))
		return
	} else {
		fmt.Fprint(w, "{\"text\": \"error\"}")
		return
	}
}

func getComments(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	news_id := vars["id"]
	if news_id == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	part := r.URL.Query().Get("part")
	if part == "" {
		part = "0"
	}

	const limit int = 10
	offset, err := strconv.ParseInt(part, 10, 0)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	offset *= int64(limit)

	rows, err := db.Query("select comments.id, comments.user_id, comments.text, comments.date, users.name, users.lastname, users.avatar from comments join users on comments.user_id=users.id where news_id=$1 and comments.deleted=false order by comments.date asc limit $2 offset $3", news_id, limit, offset)
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
	commentsCollection := []commentItem{}

	for rows.Next() {
		c := commentItem{}
		err := rows.Scan(&c.Id, &c.UserId, &c.Text, &c.Date, &c.UserName, &c.UserLastName, &c.UserAvatar)
		if err != nil {
			if err == sql.ErrNoRows {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		commentsCollection = append(commentsCollection, c)
	}

	jsonComments, err := json.Marshal(commentsCollection)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, string(jsonComments))
}

func deleteComment(w http.ResponseWriter, r *http.Request) {
	rl, err := auth(w, r)
	if err != nil {
		return
	}

	vars := mux.Vars(r)
	comm_id := vars["id"]
	if comm_id == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	_, err = db.Query("update comments set deleted=true where id=$1 and user_id=$2 and deleted=false returning id", comm_id, rl.id)
	if err != nil {
		if err == sql.ErrNoRows {
			//w.WriteHeader(http.StatusForbidden)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
