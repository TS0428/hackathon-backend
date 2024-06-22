package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

func loadEnv() {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Error loading .env file")
	}
}

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
		w.Header().Set("Access-Control-Expose-Headers", "*")
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

func getCastsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, content, filter, photo_path, video_path, user_id FROM casts")
	if err != nil {
		log.Printf("Error querying casts: %v", err)
		http.Error(w, "Failed to fetch casts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var casts []struct {
		ID        int    `json:"id"`
		Content   string `json:"content"`
		Filter    string `json:"filter"`
		PhotoPath string `json:"photoPath"`
		VideoPath string `json:"videoPath"`
		UserID    string `json:"userId"`
	}

	for rows.Next() {
		var cast struct {
			ID        int    `json:"id"`
			Content   string `json:"content"`
			Filter    string `json:"filter"`
			PhotoPath string `json:"photoPath"`
			VideoPath string `json:"videoPath"`
			UserID    string `json:"userId"`
		}

		if err := rows.Scan(&cast.ID, &cast.Content, &cast.Filter, &cast.PhotoPath, &cast.VideoPath, &cast.UserID); err != nil {
			log.Printf("Error scanning cast: %v", err)
			http.Error(w, "Failed to fetch casts", http.StatusInternalServerError)
			return
		}

		casts = append(casts, cast)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(casts); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Failed to fetch casts", http.StatusInternalServerError)
	}
}

func saveFile(fileHeader *multipart.FileHeader, destDir string) (string, error) {
	src, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	filePath := filepath.Join(destDir, fileHeader.Filename)
	dst, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}

	return filePath, nil
}

func castHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Could not parse multipart form", http.StatusBadRequest)
		return
	}

	content := r.FormValue("content")
	filter := r.FormValue("filter")
	userJson := r.FormValue("user")
	var user struct {
		DisplayName  string `json:"displayName"`
		FavoriteTeam string `json:"favoriteTeam"`
	}
	if err := json.Unmarshal([]byte(userJson), &user); err != nil {
		http.Error(w, "Invalid user data", http.StatusBadRequest)
		return
	}

	var photoPath, videoPath string
	if fileHeader, ok := r.MultipartForm.File["photo"]; ok && len(fileHeader) > 0 {
		var err error
		photoPath, err = saveFile(fileHeader[0], "./uploads")
		if err != nil {
			http.Error(w, "Could not save photo", http.StatusInternalServerError)
			return
		}
	}

	if fileHeader, ok := r.MultipartForm.File["video"]; ok && len(fileHeader) > 0 {
		var err error
		videoPath, err = saveFile(fileHeader[0], "./uploads")
		if err != nil {
			http.Error(w, "Could not save video", http.StatusInternalServerError)
			return
		}
	}

	result, err := db.Exec("INSERT INTO casts (user_id, content, filter, photo_path, video_path) VALUES (?, ?, ?, ?, ?)",
		user.DisplayName, content, filter, photoPath, videoPath)
	if err != nil {
		log.Printf("Error saving cast: %v", err)
		http.Error(w, "Failed to save cast", http.StatusInternalServerError)
		return
	}

	castID, err := result.LastInsertId()
	if err != nil {
		log.Printf("Error getting cast ID: %v", err)
		http.Error(w, "Failed to retrieve cast ID", http.StatusInternalServerError)
		return
	}

	log.Printf("Cast saved successfully for user: %v with cast ID: %d", user.DisplayName, castID)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Cast saved successfully with ID: %d", castID)))
}

func main() {
	loadEnv()
	initDB()

	http.Handle("/complete-profile", enableCORS(http.HandlerFunc(completeProfileHandler)))
	http.Handle("/casts", enableCORS(http.HandlerFunc(castHandler)))
	http.Handle("/get-casts", enableCORS(http.HandlerFunc(getCastsHandler))) // ここを追加

	log.Println("Server started at :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
