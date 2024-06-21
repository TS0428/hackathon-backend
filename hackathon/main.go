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

var db *sql.DB

func initDB() {
	// DB接続のための準備
	mysqlUser := os.Getenv("MYSQL_USER")
	mysqlPwd := os.Getenv("MYSQL_PWD")
	mysqlHost := os.Getenv("MYSQL_HOST")
	mysqlDatabase := os.Getenv("MYSQL_DATABASE")

	connStr := fmt.Sprintf("%s:%s@tcp(%s)/%s", mysqlUser, mysqlPwd, mysqlHost, mysqlDatabase)
	var err error
	db, err = sql.Open("mysql", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Connected to database")
}

func signupHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if len(req.Password) < 6 {
		log.Printf("Password too short: %v", req.Password)
		http.Error(w, "Password must be at least 6 characters", http.StatusBadRequest)
		return
	}

	_, err := db.Exec("INSERT INTO users (email, password) VALUES (?, ?)", req.Email, req.Password)
	if err != nil {
		log.Printf("Error inserting user: %v", err)
		http.Error(w, "Failed to register user", http.StatusInternalServerError)
		return
	}

	log.Printf("User registered successfully: %v", req.Email)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("User registered successfully"))
}

func completeProfileHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		TeamID   string `json:"team_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if len(req.Username) < 3 {
		log.Printf("Username too short: %v", req.Username)
		http.Error(w, "Username must be at least 3 characters", http.StatusBadRequest)
		return
	}

	_, err := db.Exec("UPDATE users SET username = ?, team_id = ? WHERE email = ?", req.Username, req.TeamID, req.Email)
	if err != nil {
		log.Printf("Error updating user profile: %v", err)
		http.Error(w, "Failed to complete profile", http.StatusInternalServerError)
		return
	}

	log.Printf("Profile completed successfully for: %v", req.Email)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Profile completed successfully"))
}

func main() {
	initDB()

	http.HandleFunc("/signup", signupHandler)
	http.HandleFunc("/complete-profile", completeProfileHandler)

	log.Println("Server started at :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
