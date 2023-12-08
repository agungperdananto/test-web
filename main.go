package main

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// User model
type User struct {
	gorm.Model
	Username string `gorm:"unique"`
	Password string
	Messages []Message
}

// Message model
type Message struct {
	gorm.Model
	Text     string
	UserID   uint
	UserName string // To store the username for simplicity in templates
}

func main() {
	// Connect to SQLite database
	db, err := gorm.Open(sqlite.Open("mydb.db"), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to the database: " + err.Error())
	}

	// Auto-migrate the User and Message models
	db.AutoMigrate(&User{}, &Message{})

	// Initialize Gin router
	router := gin.Default()

	// Set up HTML template rendering
	router.SetHTMLTemplate(template.Must(template.ParseGlob("templates/*")))

	// Set up static files serving
	router.Static("/static", "./static")

	// Define routes
	router.GET("/", func(c *gin.Context) {
		userID, err := c.Cookie("user_id")
		if err == nil && userID != "" {
			c.Redirect(http.StatusSeeOther, "/dashboard")
			return
		}

		c.HTML(http.StatusOK, "index.html", nil)
	})

	router.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", nil)
	})

	router.POST("/login", func(c *gin.Context) {
		username := c.PostForm("username")
		password := c.PostForm("password")

		var user User
		if err := db.Where("username = ? AND password = ?", username, password).First(&user).Error; err != nil {
			c.HTML(http.StatusUnauthorized, "login.html", gin.H{"Error": "Invalid credentials"})
			return
		}

		c.SetCookie("user_id", fmt.Sprint(user.ID), 3600, "/", "localhost", false, true)
		c.Redirect(http.StatusSeeOther, "/dashboard")
	})

	router.GET("/register", func(c *gin.Context) {
		c.HTML(http.StatusOK, "register.html", nil)
	})

	router.POST("/register", func(c *gin.Context) {
		username := c.PostForm("username")
		password := c.PostForm("password")

		// Check if the username already exists
		var existingUser User
		if err := db.Where("username = ?", username).First(&existingUser).Error; err == nil {
			c.HTML(http.StatusBadRequest, "register.html", gin.H{"Error": "Username already exists"})
			return
		}

		newUser := User{Username: username, Password: password}
		db.Create(&newUser)

		c.Redirect(http.StatusSeeOther, "/login")
	})

	router.GET("/dashboard", func(c *gin.Context) {
		userID, err := c.Cookie("user_id")
		if err != nil {
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}

		var messages []Message
		db.Where("user_id = ?", userID).Find(&messages)

		c.HTML(http.StatusOK, "dashboard.html", gin.H{"Messages": messages})
	})

	router.POST("/post-message", func(c *gin.Context) {
		userID, err := c.Cookie("user_id")
		if err != nil {
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}

		var user User
		if err := db.First(&user, userID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"Error": "User not found"})
			return
		}

		text := c.PostForm("text")
		message := Message{Text: text, UserID: user.ID, UserName: user.Username}
		db.Create(&message)

		c.Redirect(http.StatusSeeOther, "/dashboard")
	})

	router.GET("/logout", func(c *gin.Context) {
		c.SetCookie("user_id", "", -1, "/", "localhost", false, true)
		c.Redirect(http.StatusSeeOther, "/")
	})

	// Run the application
	router.Run(":8080")
}
