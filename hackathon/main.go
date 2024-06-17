package main

import (
	"context"
	"log"
	"net/http"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/option"
)

var client *firestore.Client

func main() {
	r := gin.Default()

	// Initialize Firebase
	opt := option.WithCredentialsFile("path/to/your/serviceAccountKey.json")
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatalf("error initializing app: %v\n", err)
	}

	auth, err := app.Auth(context.Background())
	if err != nil {
		log.Fatalf("error getting Auth client: %v\n", err)
	}

	client, err = app.Firestore(context.Background())
	if err != nil {
		log.Fatalf("error getting Firestore client: %v\n", err)
	}
	defer client.Close()

	// Routes
	r.POST("/signup", handleSignup)
	r.POST("/login", handleLogin)
	r.POST("/login/google", handleGoogleLogin)
	r.POST("/post", handleCreatePost)
	r.POST("/like", handleLikePost)
	r.POST("/retweet", handleRetweetPost)
	r.POST("/favorite-team", handleFavoriteTeam) // 新しいルート

	r.Run(":8080")
}

func handleSignup(c *gin.Context) {
	var request struct {
		UserID string `json:"user_id"`
		TeamID string `json:"team_id"`
	}
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userRef := client.Collection("users").Doc(request.UserID)
	_, err := userRef.Update(context.Background(), []firestore.Update{
		{Path: "retweet", Value: request.UserID},
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Retweet Updated"})
}

func handleLogin(c *gin.Context) {
	var request struct {
		UserID string `json:"user_id"`
		TeamID string `json:"team_id"`
	}
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userRef := client.Collection("users").Doc(request.UserID)
	_, err := userRef.Update(context.Background(), []firestore.Update{
		{Path: "login", Value: request.UserID},
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "login Updated"})
}

func handleGoogleLogin(c *gin.Context) {
	var request struct {
		UserID string `json:"user_id"`
		TeamID string `json:"team_id"`
	}
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userRef := client.Collection("users").Doc(request.UserID)
	_, err := userRef.Update(context.Background(), []firestore.Update{
		{Path: "login/google", Value: request.UserID},
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "login/google Updated"})
}

func handleCreatePost(c *gin.Context) {
	var request struct {
		UserID string `json:"user_id"`
		TeamID string `json:"team_id"`
	}
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userRef := client.Collection("users").Doc(request.UserID)
	_, err := userRef.Update(context.Background(), []firestore.Update{
		{Path: "post", Value: request.UserID},
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "post Updated"})
}

func handleLikePost(c *gin.Context) {
	var request struct {
		UserID string `json:"user_id"`
		TeamID string `json:"team_id"`
	}
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userRef := client.Collection("users").Doc(request.UserID)
	_, err := userRef.Update(context.Background(), []firestore.Update{
		{Path: "Like", Value: request.UserID},
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Like Updated"})
}

func handleRetweetPost(c *gin.Context) {
	var request struct {
		UserID string `json:"user_id"`
		TeamID string `json:"team_id"`
	}
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userRef := client.Collection("users").Doc(request.UserID)
	_, err := userRef.Update(context.Background(), []firestore.Update{
		{Path: "retweet", Value: request.UserID},
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Retweet Updated"})
}

func handleFavoriteTeam(c *gin.Context) {
	var request struct {
		UserID string `json:"user_id"`
		TeamID string `json:"team_id"`
	}
	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userRef := client.Collection("users").Doc(request.UserID)
	_, err := userRef.Update(context.Background(), []firestore.Update{
		{Path: "favorite_team", Value: request.TeamID},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Favorite team updated"})
}
