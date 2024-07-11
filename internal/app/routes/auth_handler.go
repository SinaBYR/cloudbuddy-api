package routes

import (
	"cloudbuddy/internal/pkg"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	cl "github.com/ostafen/clover/v2"
	"github.com/ostafen/clover/v2/document"
	q "github.com/ostafen/clover/v2/query"
	"golang.org/x/crypto/bcrypt"
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

		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie("Authorization", token, 3600*24*7, "", "", false, true)

		c.JSON(http.StatusCreated, gin.H{
			"uuid":       newUserId,
			"username":   newUser.Get("username"),
			"created_at": newUser.Get("created_at"),
			"token":      token,
		})
	}
}

func Signin(db *cl.DB) func(c *gin.Context) {
	return func(c *gin.Context) {
		var credentials struct {
			Username   string `json:"username"`
			Passphrase string `json:"passphrase"`
		}

		err := c.BindJSON(&credentials)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
			return
		}

		if credentials.Username == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "username is required",
			})
			return
		}
		if credentials.Passphrase == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "passphrase is required",
			})
			return
		}
		// validate password
		if len(credentials.Passphrase) < 8 {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "passphrase must be at least 8 characters long",
			})
			return
		}

		user, err := db.FindFirst(q.NewQuery("users").Where(q.Field("username").Eq(credentials.Username)))

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "An error occurred on the server",
			})
			return
		}

		if user == nil {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "username or passphrase is incorrect",
			})
			return
		}

		match, err := pkg.CheckHashPassword(credentials.Passphrase, user.Get("passphrase").(string))
		if err != nil {
			if err == bcrypt.ErrMismatchedHashAndPassword {
				c.JSON(http.StatusNotFound, gin.H{
					"message": "username or password is incorrect",
				})
				return
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": "An error occurred on the server",
				})
				return
			}
		}

		if !match {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "username or passphrase is incorrect",
			})
			return
		}

		token, err := pkg.GenerateJwtToken(user.Get("_id").(string))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "An error occurred while generating jwt token",
			})
			return
		}

		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie("Authorization", token, 3600*24*7, "", "", false, true)

		c.JSON(http.StatusCreated, gin.H{
			"uuid":       user.Get("_id").(string),
			"username":   user.Get("username"),
			"fullname":   user.Get("fullname"),
			"created_at": user.Get("created_at"),
			"token":      token,
		})
	}
}
