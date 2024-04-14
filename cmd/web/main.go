package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"database/sql"

	"github.com/gorilla/mux"

	"github.com/go-redis/redis/v8"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Config struct {
	Server struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	} `json:"server"`
	Db struct {
		User     string `json:"user"`
		Password string `json:"password"`
		Dbname   string `json:"dbname"`
	} `json:"db"`
	S3 struct {
		Host            string `json:"host"`
		AccessKeyID     string `json:"accessKeyID"`
		SecretAccessKey string `json:"secretAccessKey"`
		UseSSL          bool   `json:"useSSL"`
		Buckets         struct {
			Photo string `json:"photo"`
		} `json:"buckets"`
	} `json:"S3"`
}

var ctx context.Context
var db *sql.DB
var rdb *redis.Client
var minioClient *minio.Client

var config Config

func initDB(c Config) {
	connStr := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", c.Db.User, c.Db.Password, c.Db.Dbname)
	pg, err := sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}
	db = pg

	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	_, err = rdb.Ping(ctx).Result()
	if err != nil {
		panic(err)
	}
}

func initS3(c Config) {
	var err error
	minioClient, err = minio.New(c.S3.Host, &minio.Options{
		Creds:  credentials.NewStaticV4(c.S3.AccessKeyID, c.S3.SecretAccessKey, ""),
		Secure: c.S3.UseSSL,
	})

	if err != nil {
		log.Fatalln(err)
	}
}

func main() {
	ctx = context.Background()

	f, err := os.Open("config.json")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(&config)
	if err != nil {
		panic(err)
	}

	fs := http.FileServer(http.Dir("./ui/static"))
	http.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")

		http.StripPrefix("/static/", fs).ServeHTTP(w, r)
	})

	router := mux.NewRouter()

	router.HandleFunc("/uploads/{id}", servStorage).Methods("GET")

	router.HandleFunc("/api/signin", Signin)
	router.HandleFunc("/api/signup", Signup)
	router.HandleFunc("/api/logout", Logout)
	router.HandleFunc("/api/ping", Ping)

	router.HandleFunc("/api/users", getUsers).Methods("GET")
	router.HandleFunc("/api/users/{id:[0-9]+}", getUser).Methods("GET")

	router.HandleFunc("/api/user", partUpdateUser).Methods("PATCH")
	router.HandleFunc("/api/user", deleteUser).Methods("DELETE")

	router.HandleFunc("/api/user/avatar", updateUserAvatar).Methods("POST")
	router.HandleFunc("/api/user/avatar", deleteUserAvatar).Methods("DELETE")

	router.HandleFunc("/api/news", addNews).Methods("POST")
	router.HandleFunc("/api/news/{id:[0-9]+}", deleteNews).Methods("DELETE")
	// router.HandleFunc("/api/news/{id:[0-9]+}", partUpdateNews).Methods("PATCH")

	router.HandleFunc("/api/news/user/{id:[0-9]+}", getNews).Methods("GET")

	router.HandleFunc("/api/comments", addComment).Methods("POST")
	router.HandleFunc("/api/comments/{id:[0-9]+}", deleteComment).Methods("DELETE")
	// router.HandleFunc("/api/comments/{id:[0-9]+}", partUpdateComment).Methods("PATCH")

	router.HandleFunc("/api/comments/news/{id:[0-9]+}", getComments).Methods("GET")

	router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./ui/static/index.html")
	})

	http.Handle("/", router)

	initDB(config)
	initS3(config)

	fmt.Println("Server is listening...")
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port), nil))
}
