package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

// User struct represents the data structure for user information
type User struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	TeamID   string `json:"team_id"`
}

// Post struct represents the data structure for post information
type Post struct {
	ID       int    `json:"id"`
	UserID   int    `json:"user_id"`
	Content  string `json:"content"`
	PostedAt string `json:"posted_at"`
}

// Database connection
var db *sql.DB

func initDB() {
	// MySQL connection
	mysqlUser := os.Getenv("MYSQL_USER")
	mysqlUserPwd := os.Getenv("MYSQL_PASSWORD")
	mysqlDatabase := os.Getenv("MYSQL_DATABASE")

	dataSourceName := fmt.Sprintf("%s:%s@tcp(localhost:3306)/%s", mysqlUser, mysqlUserPwd, mysqlDatabase)
	var err error
	db, err = sql.Open("mysql", dataSourceName)
	if err != nil {
		log.Fatalf("error connecting to MySQL: %v\n", err)
	}
}

func main() {
	initDB()
	defer db.Close()

	http.HandleFunc("/signup", cors(handleSignup))
	http.HandleFunc("/login", cors(handleLogin))
	http.HandleFunc("/posts", cors(handleCreatePost))
	http.HandleFunc("/posts/all", cors(handleGetPosts))

	fmt.Println("Server running on localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// cors is a middleware function that handles CORS headers for all endpoints
func cors(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		handler(w, r)
	}
}

func handleSignup(w http.ResponseWriter, r *http.Request) {
	log.Printf("Request method: %s\n", r.Method)
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Error decoding request body: "+err.Error(), http.StatusBadRequest)
		log.Println("Error decoding request body:", err)
		return
	}

	log.Printf("Received user: %+v\n", user)

	// Insert user into MySQL database
	stmt, err := db.Prepare("INSERT INTO users (username, email, password, team_id) VALUES (?, ?, ?, ?)")
	if err != nil {
		http.Error(w, "Error preparing SQL statement: "+err.Error(), http.StatusInternalServerError)
		log.Println("Error preparing SQL statement:", err)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(user.Username, user.Email, user.Password, user.TeamID)
	if err != nil {
		http.Error(w, "Error executing SQL statement: "+err.Error(), http.StatusInternalServerError)
		log.Println("Error executing SQL statement:", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "User registered successfully")
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	log.Printf("Request method: %s\n", r.Method)
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Error decoding request body: "+err.Error(), http.StatusBadRequest)
		log.Println("Error decoding request body:", err)
		return
	}

	// MySQLデータベースからユーザーを検索
	stmt, err := db.Prepare("SELECT username, email, team_id FROM users WHERE username = ? AND password = ?")
	if err != nil {
		http.Error(w, "Error preparing SQL statement: "+err.Error(), http.StatusInternalServerError)
		log.Println("Error preparing SQL statement:", err)
		return
	}
	defer stmt.Close()

	var dbUser User
	err = stmt.QueryRow(user.Username, user.Password).Scan(&dbUser.Username, &dbUser.Email, &dbUser.TeamID)
	if err == sql.ErrNoRows {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	} else if err != nil {
		http.Error(w, "Error executing SQL statement: "+err.Error(), http.StatusInternalServerError)
		log.Println("Error executing SQL statement:", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(dbUser)
}

func handleCreatePost(w http.ResponseWriter, r *http.Request) {
	log.Printf("Request method: %s\n", r.Method)
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var post Post
	err := json.NewDecoder(r.Body).Decode(&post)
	if err != nil {
		http.Error(w, "Error decoding request body: "+err.Error(), http.StatusBadRequest)
		log.Println("Error decoding request body:", err)
		return
	}

	stmt, err := db.Prepare("INSERT INTO posts (user_id, content) VALUES (?, ?)")
	if err != nil {
		http.Error(w, "Error preparing SQL statement: "+err.Error(), http.StatusInternalServerError)
		log.Println("Error preparing SQL statement:", err)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(post.UserID, post.Content)
	if err != nil {
		http.Error(w, "Error executing SQL statement: "+err.Error(), http.StatusInternalServerError)
		log.Println("Error executing SQL statement:", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Post created successfully")
}

func handleGetPosts(w http.ResponseWriter, r *http.Request) {
	log.Printf("Request method: %s\n", r.Method)
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	rows, err := db.Query("SELECT id, user_id, content, posted_at FROM posts")
	if err != nil {
		http.Error(w, "Error querying posts: "+err.Error(), http.StatusInternalServerError)
		log.Println("Error querying posts:", err)
		return
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		err := rows.Scan(&post.ID, &post.UserID, &post.Content, &post.PostedAt)
		if err != nil {
			http.Error(w, "Error scanning post: "+err.Error(), http.StatusInternalServerError)
			log.Println("Error scanning post:", err)
			return
		}
		posts = append(posts, post)
	}

	if err = rows.Err(); err != nil {
		http.Error(w, "Error reading posts: "+err.Error(), http.StatusInternalServerError)
		log.Println("Error reading posts:", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(posts)
}
