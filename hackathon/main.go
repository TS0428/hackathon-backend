package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"

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
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
}

type UserRegisterReq struct {
	Id       int    `json:"id"`
	UserName string `json:"user_name"`
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
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		_, err = db.Query("INSERT INTO users (user_name, email) VALUES (?, ?)", userReq.UserName, userReq.Email)
		if err != nil {
			log.Printf("fail: insert user, %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	default:
		log.Printf("fail: HTTP Method is %s\n", r.Method)
		http.Error(w, "Invalid HTTP Method", http.StatusMethodNotAllowed)
	}
}

func userSelectHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w, r)

	switch r.Method {
	case http.MethodGet:
		UserName := r.URL.Query().Get("user_name")
		Email := r.URL.Query().Get("email")

		if UserName == "" || Email == "" {
			log.Printf("missing query parameters")
			http.Error(w, "missing query parameters", http.StatusBadRequest)
			return
		}

		var user UserRegisterReq
		err := db.QueryRow("SELECT id, user_name FROM users WHERE user_name = ? AND email = ?", UserName, Email).Scan(&user.Id, &user.UserName)
		if err != nil {
			log.Printf("fail: select user, %v\n", err)
			http.Error(w, "Failed to select user", http.StatusInternalServerError)
			return
		}
		response, err := json.Marshal(user)
		if err != nil {
			log.Printf("fail: json marshal response, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(response)
		if err != nil {
			log.Printf("fail: write response, %v\n", err)
		}
	default:
		log.Printf("fail: HTTP Method is %s\n", r.Method)
		http.Error(w, "Invalid HTTP Method", http.StatusMethodNotAllowed)
	}
}

type TweetReq struct {
	Id       int     `json:"id"`
	UserId   int     `json:"user_id"`
	Content  string  `json:"content"`
	Likes    int     `json:"likes"`
	Replies  []Reply `json:"replies"`
	UserName string  `json:"user_name"`
}

type Reply struct {
	Id      int    `json:"id"`
	UserId  int    `json:"user_id"`
	TweetId int    `json:"tweet_id"`
	Content string `json:"content"`
}

func tweetHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w, r)
	switch r.Method {
	case http.MethodPost:
		var tweetReq TweetReq
		err := json.NewDecoder(r.Body).Decode(&tweetReq)
		if err != nil {
			log.Printf("err in decode cast: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("Failed to decode JSON: %v", err)))
			return
		}

		repliesJSON, err := json.Marshal(tweetReq.Replies)
		if err != nil {
			log.Printf("fail: json.Marshal replies, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, err = db.Query("INSERT INTO tweets (user_id, content, likes, replies, user_name) VALUES (?, ?, ?, ?, ?)", tweetReq.UserId, tweetReq.Content, tweetReq.Likes, repliesJSON, tweetReq.UserName)
		if err != nil {
			log.Printf("fail: insert tweet, %v\n", err)
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

func tweetShowHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w, r)
	switch r.Method {
	case http.MethodGet:

		rows, err := db.Query("SELECT id, user_name, user_id, content, replies FROM tweets")
		if err != nil {
			log.Printf("fail: db.Query, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		defer func() {
			if err := rows.Close(); err != nil {
				log.Printf("fail: rows.Close, %v\n", err)
			}
		}()

		var tweets []TweetReq
		for rows.Next() {
			var tweet TweetReq
			var repliesJSON []byte
			if err := rows.Scan(&tweet.Id, &tweet.UserName, &tweet.UserId, &tweet.Content, &tweet.Replies); err != nil {
				log.Printf("fail: rows.Scan, %v\n", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if len(repliesJSON) > 0 {
				var replies []Reply
				if err := json.Unmarshal(repliesJSON, &replies); err != nil {
					log.Printf("fail: json.Unmarshal, %v\n", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				tweet.Replies = replies
			} else {
				tweet.Replies = []Reply{}
			}

			tweets = append(tweets, tweet)
		}

		if err := rows.Err(); err != nil {
			log.Printf("fail: rows.Err, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		bytes, err := json.Marshal(tweets)
		if err != nil {
			log.Printf("fail: json.Marshal, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(bytes)
	default:
		log.Printf("fail: HTTP Method is %s\n", r.Method)
		w.WriteHeader(http.StatusBadRequest)
	}
}

func repliesHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w, r)
	switch r.Method {
	case http.MethodPost:
		var repliesReq Reply
		err := json.NewDecoder(r.Body).Decode(&repliesReq)
		if err != nil {
			log.Printf("err in decode cast")
			w.WriteHeader(http.StatusBadRequest)
		}

		_, err = db.Query("INSERT INTO replies (user_id, tweet_id, content) VALUES (?, ?, ?)", repliesReq.UserId, repliesReq.TweetId, repliesReq.Content)

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

type LikesReq struct {
	Id      int `json:"id"`
	UserId  int `json:"user_id"`
	TweetId int `json:"tweet_id"`
}

type LikesCount struct {
	TweetId int `json:"tweet_id"`
	Count   int `json:"count"`
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

		_, err = db.Query("INSERT INTO likes (user_id, cast_id) VALUES (?, ?)", likesReq.UserId, likesReq.TweetId)

		if err != nil {
			log.Printf("fail: post likes, %v\n", err)
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

func likesCountHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w, r)
	switch r.Method {
	case http.MethodGet:
		rows, err := db.Query("SELECT tweet_id, COUNT(*) as count FROM likes GROUP BY tweet_id")
		if err != nil {
			log.Printf("fail: db.Query, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var likesCounts []LikesCount
		for rows.Next() {
			var likesCount LikesCount
			if err := rows.Scan(&likesCount.TweetId, &likesCount.Count); err != nil {
				log.Printf("fail: rows.Scan, %v\n", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			likesCounts = append(likesCounts, likesCount)
		}

		if err := rows.Err(); err != nil {
			log.Printf("fail: rows.Err, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		response, err := json.Marshal(likesCounts)
		if err != nil {
			log.Printf("fail: json.Marshal, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(response)
	default:
		log.Printf("fail: HTTP Method is %s\n", r.Method)
		w.WriteHeader(http.StatusBadRequest)
	}
}

func main() {
	loadEnv()
	initDB()

	http.HandleFunc("/user", userHandler)
	http.HandleFunc("/user/select", userSelectHandler)
	http.HandleFunc("/tweets", tweetHandler)
	http.HandleFunc("/tweets/show", tweetShowHandler)
	http.HandleFunc("/likes", likesHandler)
	http.HandleFunc("/likes/count", likesCountHandler)
	http.HandleFunc("/replies", repliesHandler)

	log.Println("Server started at :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
