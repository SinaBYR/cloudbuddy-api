package routes

import (
	"cloudbuddy/internal/pkg"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	cl "github.com/ostafen/clover/v2"
	"github.com/ostafen/clover/v2/document"
	q "github.com/ostafen/clover/v2/query"
)

func Signup(db *cl.DB) func(c *gin.Context) {
	return func(c *gin.Context) {
		username := c.PostForm("username")
		passphrase := c.PostForm("passphrase")

		if username == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "username is required",
			})
			return
		}
		if passphrase == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "passphrase is required",
			})
			return
		}
		// validate password
		if len(passphrase) < 8 {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "passphrase must be at least 8 characters long",
			})
			return
		}

		doc, err := db.FindFirst(q.NewQuery("users").Where(q.Field("username").Eq(username)))

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "An error occurred on the server",
			})
			return
		}

		if doc != nil {
			c.JSON(http.StatusConflict, gin.H{
				"message": "A user with that username already exists",
			})
			return
		}

		// hash password
		hashedPassphrase, err := pkg.HashPassword(passphrase)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "An error occurred while hashing passphrase",
			})
			return
		}

		newUser := document.NewDocument()
		newUser.Set("username", username)
		newUser.Set("passphrase", hashedPassphrase)
		newUser.Set("created_at", time.Now())

		newUserId, err := db.InsertOne("users", newUser)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "An error occurred while creating a new user",
			})
			return
		}

		token, err := pkg.GenerateJwtToken(newUserId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "An error occurred while generating jwt token",
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"uuid":       newUserId,
			"username":   newUser.Get("username"),
			"created_at": newUser.Get("created_at"),
			"token":      token,
		})
	}
}
