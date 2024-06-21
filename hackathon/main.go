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

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func completeProfileHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDToken  string `json:"idToken"`
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

	if len(req.ID) < 3 {
		log.Printf("ID too short: %v", req.ID)
		http.Error(w, "ID must be at least 3 characters", http.StatusBadRequest)
		return
	}

	_, err := db.Exec("INSERT INTO users (id_token, username, id, team_id) VALUES (?, ?, ?, ?)", req.IDToken, req.Username, req.ID, req.TeamID)
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

	http.Handle("/complete-profile", enableCORS(http.HandlerFunc(completeProfileHandler)))

	log.Println("Server started at :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
