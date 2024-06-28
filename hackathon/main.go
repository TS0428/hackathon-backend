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
	Id       int    `json:"id"`
	UserId   int    `json:"user_id"`
	TweetId  int    `json:"tweet_id"`
	Content  string `json:"content"`
	UserName string `json:"user_name"`
}

func tweetHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w, r)
	switch r.Method {
	case http.MethodPost:
		var tweetReq TweetReq
		err := json.NewDecoder(r.Body).Decode(&tweetReq)
		if err != nil {
			log.Printf("err in decode tweet: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("Failed to decode JSON: %v", err)))
			return
		}

		// replies が nil の場合は空のスライスに設定
		if tweetReq.Replies == nil {
			tweetReq.Replies = []Reply{}
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
		rows, err := db.Query("SELECT id, user_name, user_id, content, replies, likes FROM tweets")
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
			if err := rows.Scan(&tweet.Id, &tweet.UserName, &tweet.UserId, &tweet.Content, &repliesJSON, &tweet.Likes); err != nil {
				log.Printf("fail: rows.Scan, %v\n", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if len(repliesJSON) > 0 && string(repliesJSON) != "{}" {
				var replies []Reply
				log.Printf("repliesJSON: %s", string(repliesJSON))
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
			return
		}

		tx, err := db.Begin()
		if err != nil {
			log.Printf("fail: begin transaction, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Insert new reply
		_, err = tx.Exec("INSERT INTO replies (user_id, tweet_id, content, user_name) VALUES (?, ?, ?, ?)", repliesReq.UserId, repliesReq.TweetId, repliesReq.Content, repliesReq.UserName)
		if err != nil {
			tx.Rollback()
			log.Printf("fail: insert reply, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Get updated replies for the tweet
		var replies []Reply
		rows, err := tx.Query("SELECT id, user_id, tweet_id, content, user_name FROM replies WHERE tweet_id = ?", repliesReq.TweetId)
		if err != nil {
			tx.Rollback()
			log.Printf("fail: select replies, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var reply Reply
			if err := rows.Scan(&reply.Id, &reply.UserId, &reply.TweetId, &reply.Content, &reply.UserName); err != nil {
				tx.Rollback()
				log.Printf("fail: scan reply, %v\n", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			replies = append(replies, reply)
		}

		// Convert replies to JSON
		repliesJSON, err := json.Marshal(replies)
		if err != nil {
			tx.Rollback()
			log.Printf("fail: marshal replies, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Update the tweet with the new replies JSON
		_, err = tx.Exec("UPDATE tweets SET replies = ? WHERE id = ?", repliesJSON, repliesReq.TweetId)
		if err != nil {
			tx.Rollback()
			log.Printf("fail: update tweet replies, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			tx.Rollback()
			log.Printf("fail: commit transaction, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	default:
		log.Printf("fail: HTTP Method is %s\n", r.Method)
		w.WriteHeader(http.StatusBadRequest)
	}
}

type LikesReq struct {
	UserId  int `json:"user_id"`
	TweetId int `json:"tweet_id"`
}

func likesHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w, r)
	switch r.Method {
	case http.MethodPost:
		var likesReq LikesReq
		err := json.NewDecoder(r.Body).Decode(&likesReq)
		if err != nil {
			log.Printf("err in decode cast: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		tx, err := db.Begin()
		if err != nil {
			log.Printf("fail: begin transaction, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, err = tx.Exec("INSERT INTO likes (user_id, tweet_id) VALUES (?, ?)", likesReq.UserId, likesReq.TweetId)
		if err != nil {
			tx.Rollback()
			log.Printf("fail: insert into likes, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, err = tx.Exec("UPDATE tweets SET likes = likes + 1 WHERE id = ?", likesReq.TweetId)
		if err != nil {
			tx.Rollback()
			log.Printf("fail: update likes, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = tx.Commit()
		if err != nil {
			log.Printf("fail: commit transaction, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	case http.MethodOptions: // ここにOPTIONSメソッドの処理を追加
		w.WriteHeader(http.StatusOK)
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
	http.HandleFunc("/replies", repliesHandler)

	log.Println("Server started at :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
