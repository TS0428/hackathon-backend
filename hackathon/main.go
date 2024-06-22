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
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var users struct {
		Id       string `json:"id"`
		Username string `json:"username"`
		TeamID   string `json:"team_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&users); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	_, err := db.Exec("UPDATE users SET username = ?, team_id = ? WHERE id = ?", users.Username, users.TeamID, users.Id)
	if err != nil {
		log.Printf("Error updating user profile: %v", err)
		http.Error(w, "Failed to update user profile", http.StatusInternalServerError)
		return
	}

	log.Printf("User profile updated successfully: %s", users.Username)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("User profile updated successfully: %s", users.Username)))
}

func getCastsHandler(w http.ResponseWriter, r *http.Request) {
	favoriteTeam := r.URL.Query().Get("favoriteTeam")
	var rows *sql.Rows
	var err error

	if favoriteTeam != "" {
		rows, err = db.Query(`
			SELECT c.id, c.content, c.filter, c.photo_path, c.video_path, u.username, u.team_id
			FROM casts c
			JOIN users u ON c.user_id = u.id
			WHERE u.favoriteTeam = ?`, favoriteTeam)
	} else {
		rows, err = db.Query(`
			SELECT c.id, c.content, c.filter, c.photo_path, c.video_path, u.username, u.team_id
			FROM casts c
			JOIN users u ON c.user_id = u.id`)
	}

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
		User      struct {
			DisplayName  string `json:"displayName"`
			FavoriteTeam string `json:"favoriteTeam"`
		} `json:"user"`
	}

	for rows.Next() {
		var cast struct {
			ID        int    `json:"id"`
			Content   string `json:"content"`
			Filter    string `json:"filter"`
			PhotoPath string `json:"photoPath"`
			VideoPath string `json:"videoPath"`
			User      struct {
				DisplayName  string `json:"displayName"`
				FavoriteTeam string `json:"favoriteTeam"`
			} `json:"user"`
		}

		if err := rows.Scan(&cast.ID, &cast.Content, &cast.Filter, &cast.PhotoPath, &cast.VideoPath, &cast.User.DisplayName, &cast.User.FavoriteTeam); err != nil {
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

func likeCastHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var requestData struct {
		ID int `json:"id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	_, err := db.Exec(`UPDATE casts SET likes = likes + 1 WHERE id = ?`, requestData.ID)
	if err != nil {
		log.Printf("Error updating likes: %v", err)
		http.Error(w, "Failed to update likes", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func replyCastHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var requestData struct {
		ID      int    `json:"id"`
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	var replies []string
	err := db.QueryRow(`SELECT replies FROM casts WHERE id = ?`, requestData.ID).Scan(&replies)
	if err != nil {
		log.Printf("Error fetching replies: %v", err)
		http.Error(w, "Failed to fetch replies", http.StatusInternalServerError)
		return
	}

	replies = append(replies, requestData.Content)
	repliesJSON, err := json.Marshal(replies)
	if err != nil {
		log.Printf("Error marshalling replies: %v", err)
		http.Error(w, "Failed to update replies", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec(`UPDATE casts SET replies = ? WHERE id = ?`, repliesJSON, requestData.ID)
	if err != nil {
		log.Printf("Error updating replies: %v", err)
		http.Error(w, "Failed to update replies", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
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
	http.Handle("/get-casts", enableCORS(http.HandlerFunc(getCastsHandler)))   // ここを追加
	http.Handle("/like-cast", enableCORS(http.HandlerFunc(likeCastHandler)))   // CORSを有効にする
	http.Handle("/reply-cast", enableCORS(http.HandlerFunc(replyCastHandler))) // CORSを有効にする

	log.Println("Server started at :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
