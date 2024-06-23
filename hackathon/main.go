package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

func loadEnv() {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Error loading .env file")
	}
} //

func initDB() {
	mysqlUser := os.Getenv("MYSQL_USER")
	mysqlPwd := os.Getenv("MYSQL_PWD")
	mysqlHost := os.Getenv("MYSQL_HOST")
	mysqlDatabase := os.Getenv("MYSQL_DATABASE")

	log.Printf("MYSQL_USER: %s", mysqlUser)
	log.Printf("MYSQL_PWD: %s", mysqlPwd)
	log.Printf("MYSQL_HOST: %s", mysqlHost)
	log.Printf("MYSQL_DATABASE: %s", mysqlDatabase)

	connStr := fmt.Sprintf("%s:%s@%s/%s", mysqlUser, mysqlPwd, mysqlHost, mysqlDatabase)

	_db, err := sql.Open("mysql", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := _db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	db = _db

	log.Println("Connected to database")
}

func enableCORS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Expose-Headers", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
}

type UserRegisterReq struct {
	UserName string `json:"username"`
	Email    string `json:"email"`
}

func userHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w, r)
	switch r.Method {
	case http.MethodPost:
		var userReq UserRegisterReq
		err := json.NewDecoder(r.Body).Decode(&userReq)
		if err != nil {
			log.Printf("err in decode user")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		_, err = db.Query("INSERT INTO users (username, email) VALUES ( ?, ?)", userReq.UserName, userReq.Email)
		if err != nil {
			log.Printf("fail: insert user, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		return
	default:
		log.Printf("fail: HTTP Method is %s\n", r.Method)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

}

type CastReq struct {
	UserId  int    `json:"userid"`
	Content string `json:"content"`
	Likes   int    `json:"likes"`
	Replies string `json:"replies"`
}

func castHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w, r)
	switch r.Method {
	case http.MethodPost:
		var castReq CastReq
		err := json.NewDecoder(r.Body).Decode(&castReq)
		if err != nil {
			log.Printf("err in decode cast")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if castReq.Replies == "" {
			castReq.Replies = "{}"
		}

		_, err = db.Query("INSERT INTO casts (userid, content, likes, replies) VALUES (?, ?, ?, ?)", castReq.UserId, castReq.Content, castReq.Likes, castReq.Replies)
		if err != nil {
			log.Printf("fail: insert cast, %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		return
	default:
		log.Printf("fail: HTTP Method is %s\n", r.Method)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

type LikesReq struct {
	UserId int `json:"user_id"`
	CastId int `json:"cast_id"`
}

func likesHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w, r)
	switch r.Method {
	case http.MethodPost:
		var likesReq LikesReq
		err := json.NewDecoder(r.Body).Decode(&likesReq)
		if err != nil {
			log.Printf("err in decode cast")
			w.WriteHeader(http.StatusBadRequest)
		}

		_, err = db.Query("INSERT INTO likes (user_id, cast_id) VALUES (?, ?)", likesReq.UserId, likesReq.CastId)

		if err != nil {
			log.Printf("fail: get cast, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		return
	default:
		log.Printf("fail: HTTP Method is %s\n", r.Method)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

}

type RepliesReq struct {
	UserId  int    `json:"user_id"`
	CastId  int    `json:"cast_id"`
	Content string `json:"content"`
}

func repliesHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w, r)
	switch r.Method {
	case http.MethodPost:
		var repliesReq RepliesReq
		err := json.NewDecoder(r.Body).Decode(&repliesReq)
		if err != nil {
			log.Printf("err in decode cast")
			w.WriteHeader(http.StatusBadRequest)
		}

		_, err = db.Query("INSERT INTO replies (user_id, cast_id, content) VALUES (?, ?, ?)", repliesReq.UserId, repliesReq.CastId, repliesReq.Content)

		if err != nil {
			log.Printf("fail: query replies, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		return
	default:
		log.Printf("fail: HTTP Method is %s\n", r.Method)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

}

func main() {
	loadEnv()
	initDB()

	http.HandleFunc("/user", userHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
	http.HandleFunc("/casts", castHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
	http.HandleFunc("/likes", likesHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
	http.HandleFunc("/replies", repliesHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))

	log.Println("Server started at :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
