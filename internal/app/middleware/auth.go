package middleware

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	cl "github.com/ostafen/clover/v2"
)

func DecodeJwtMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		err := godotenv.Load()
		if err != nil {
			log.Printf("Couldn't load environment variables: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"message": "An unexpected error occured on the server",
			})
			return
		}

		jwtSecret := os.Getenv("JWT_SECRET")
		if jwtSecret == "" {
			log.Printf("JWT_SECRET environment variable is not set: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"message": "An unexpected error occured on the server",
			})
			return
		}

		tokenString, err := c.Cookie("Authorization")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "unauthorized",
			})
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// validate the alg is what you expect:
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			return []byte(jwtSecret), nil
		})

		if err != nil {
			log.Printf("jwt token parsing failed: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"message": "An unexpected error occured on the server",
			})
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			// check the expiry date
			if float64(time.Now().Unix()) > claims["exp"].(float64) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"message": "unauthorized",
				})
				return
			}

			// find the user with token Subject
			db, _ := cl.Open("clover-db")
			defer db.Close()
			user, err := db.FindById("users", claims["sub"].(string))

			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"message": "unauthorized",
				})
				return
			}

			// attach the request
			c.Set("user", user)

			// continue
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"message": "unauthorized",
		})
	}
}
