package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
)

var db *sql.DB

func loadEnv() {

	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Error loading .env file")
	}
}
func initDB() {
	// DB接続のための準備
	mysqlUser := os.Getenv("MYSQL_USER")
	mysqlPwd := os.Getenv("MYSQL_PWD")
	mysqlHost := os.Getenv("MYSQL_HOST")
	mysqlDatabase := os.Getenv("MYSQL_DATABASE")

	log.Printf("MYSQL_USER: %s", mysqlUser)
	log.Printf("MYSQL_PWD: %s", mysqlPwd)
	log.Printf("MYSQL_HOST: %s", mysqlHost)
	log.Printf("MYSQL_DATABASE: %s", mysqlDatabase)

	connStr := fmt.Sprintf("%s:%s@%s/%s", mysqlUser, mysqlPwd, mysqlHost, mysqlDatabase)
	db, err := sql.Open("mysql", connStr)
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
		Id       string `json:"id"`
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
		ID       string `json:"id"`
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

	if len(req.ID) < 5 {
		log.Printf("ID too short: %v", req.ID)
		http.Error(w, "ID must be at least 5 characters", http.StatusBadRequest)
		return
	}

	_, err := db.Exec("UPDATE users SET username = ?, team_id = ? WHERE id = ?", req.Username, req.TeamID, req.ID)
	if err != nil {
		log.Printf("Error updating user profile: %v", err)
		http.Error(w, "Failed to complete profile", http.StatusInternalServerError)
		return
	}

	log.Printf("Profile completed successfully for: %v", req.ID)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Profile completed successfully"))
}

func main() {

	loadEnv()
	initDB()

	http.HandleFunc("/signup", signupHandler)
	http.HandleFunc("/complete-profile", completeProfileHandler)

	log.Println("Server started at :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
