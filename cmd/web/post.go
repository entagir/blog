package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"database/sql"

	"github.com/gorilla/mux"
	"github.com/lib/pq"
)

type news struct {
	Id       int       `json:"id"`
	Text     string    `json:"text"`
	Date     time.Time `json:"date"`
	Comments int       `json:"comments"`
	Photos   []string  `json:"photos"`
}

func getNews(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	user_id := vars["id"]

	if user_id == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	result := db.QueryRow("select deleted from users where id=$1 and deleted=false", user_id)
	var deletedUser bool
	err := result.Scan(&deletedUser)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
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

	start := r.URL.Query().Get("start")
	startId, err := strconv.ParseInt(start, 10, 0)

	var rows *sql.Rows
	if err != nil || startId == 0 {
		rows, err = db.Query("select id, text, date, photos, (select count(*) FROM comments where news_id=news.id and deleted=false) as comms from news where user_id=$1 and deleted=false order by date desc limit $2 offset $3", user_id, limit, offset)
	} else {
		rows, err = db.Query("select id, text, date, photos, (select count(*) FROM comments where news_id=news.id and deleted=false) as comms from news where user_id=$1 and id<=$4 and deleted=false order by date desc limit $2 offset $3", user_id, limit, offset, startId)
	}
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
	newsCollection := []news{}

	for rows.Next() {
		n := news{}
		err := rows.Scan(&n.Id, &n.Text, &n.Date, pq.Array(&n.Photos), &n.Comments)
		if err != nil {
			if err == sql.ErrNoRows {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		newsCollection = append(newsCollection, n)
	}

	jsonNews, err := json.Marshal(newsCollection)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, string(jsonNews))
}

func addNews(w http.ResponseWriter, r *http.Request) {
	rl, err := auth(w, r)
	if err != nil {
		return
	}

	date := time.Now().UTC()

	// Request validation

	// Post text
	text := r.FormValue("text")

	// Post photo
	files := r.MultipartForm.File["img"]
	onValidFile := false
	var photos []string

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			continue
		}
		defer file.Close()

		photoId, err := loadImage(file, rl, date, false)
		if err != nil {
			continue
		}

		photos = append(photos, photoId)
		onValidFile = true
		file.Close()
	}

	if !onValidFile && text == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	rows, err := db.Query("insert into news (user_id, text, date, photos) values ($1, $2, $3, $4) RETURNING id, text, date, photos", rl.id, text, date, pq.Array(photos))
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	defer rows.Close()
	newsCollection := []news{}

	if rows.Next() {
		n := news{}
		err = rows.Scan(&n.Id, &n.Text, &n.Date, pq.Array(&n.Photos))
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		newsCollection = append(newsCollection, n)

		jsonNews, err := json.Marshal(newsCollection)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		fmt.Fprint(w, string(jsonNews))
		return
	} else {
		fmt.Fprint(w, "{\"text\": \"error\"}")
		return
	}
}

func deleteNews(w http.ResponseWriter, r *http.Request) {
	rl, err := auth(w, r)
	if err != nil {
		return
	}

	vars := mux.Vars(r)
	news_id := vars["id"]
	if news_id == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	rows, err := db.Query("update news set deleted=true where id=$1 and user_id=$2 and deleted=false RETURNING photos", news_id, rl.id)
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
	defer rows.Close()

	if rows.Next() {
		var photos []string
		err = rows.Scan(pq.Array(&photos))
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Clean photos
		if len(photos) != 0 {
			go cleanPhotos(photos, false)
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
