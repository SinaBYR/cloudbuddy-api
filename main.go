package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	cl "github.com/ostafen/clover/v2"
	"github.com/ostafen/clover/v2/document"
	q "github.com/ostafen/clover/v2/query"
)

type Image struct {
	UUID      string    `clover:"_id"`
	Url       string    `clover:"url"`
	Likes     int64     `clover:"likes"`
	CreatedAt time.Time `clover:"created_at"`
}

func main() {
	db, _ := cl.Open("clover-db")
	defer db.Close()

	if has, _ := db.HasCollection("images"); !has {
		db.CreateCollection("images")
	}

	r := gin.Default()

	images := r.Group("/v1/images")

	images.GET("/", getAllImages(db))
	images.GET("/:id", getImageById(db))

	r.Run()
}

func getImageById(db *cl.DB) func(c *gin.Context) {
	return func(c *gin.Context) {
		id := c.Param("id")
		doc, err := db.FindFirst(q.NewQuery("images").Where(q.Field("uuid").Eq(id)))

		if doc == nil {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "cloud not found :(",
			})
			return
		}

		if err != nil {
			if err == cl.ErrDocumentNotExist {
				c.JSON(http.StatusNotFound, gin.H{
					"message": "cloud not found :(",
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": "wtf",
				})
			}

			return
		}

		c.JSON(http.StatusOK, doc)
	}
}

func getAllImages(db *cl.DB) func(c *gin.Context) {
	return func(c *gin.Context) {
		docs, err := db.FindAll(q.NewQuery("images"))

		fmt.Println(docs)

		if err != nil {
			if err == cl.ErrCollectionNotExist {
				c.JSON(http.StatusNotFound, gin.H{
					"message": err.Error(),
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": "wtf",
				})
			}

			return
		}

		c.JSON(http.StatusOK, docs)
	}
}

func postImage(db *cl.DB) func(c *gin.Context) {
	return func(c *gin.Context) {
		doc := document.NewDocument()
		doc.Set("url", "")
		doc.Set("likes", 0)
		doc.Set("created_at", time.Now())
		docId, err := db.InsertOne("images", doc)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "An error occured while creating a new image record",
			})
			return
		}

		file, err := c.FormFile("image")
		if err != nil {
			c.JSON(http.StatusNotAcceptable, gin.H{
				"message": "Image file is not acceptable",
			})
			return
		}

		file.Filename = file.Filename + docId

		err = uploadToBucket(file)
		if err != nil {
			log.Println(err)
			innerErr := db.DeleteById("images", docId)
			if innerErr != nil {
				log.Println(innerErr)
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": "An error occured while deleting temporary created image record",
				})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "An error occured while uploading image to the bucket",
			})
			return
		}

		err = godotenv.Load()
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "An error occured",
			})
			return
		}

		url := strings.Join([]string{
			os.Getenv("BUCKET_ENDPOINT"),
			os.Getenv("BUCKET_NAME"),
			"cloudbuddy",
			file.Filename,
		}, "/")

		err = db.UpdateById("images", docId, func(doc *document.Document) *document.Document {
			doc.Set("url", url)
			return doc
		})

		if err != nil {
			if err == cl.ErrDocumentNotExist {
				log.Println(err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": "Image upload failed",
				})
				return
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": "An error occured",
				})
				return
			}
		}

		c.JSON(http.StatusCreated, doc)
	}
}

func uploadToBucket(file *multipart.FileHeader) error {
	err := godotenv.Load()
	if err != nil {
		return errors.New(fmt.Sprintf("Error loading environment variables: %s", err.Error()))
	}

	accessKey := os.Getenv("BUCKET_ACCESS_KEY")
	secretKey := os.Getenv("BUCKET_SECRET_KEY")
	bucketName := os.Getenv("BUCKET_NAME")

	if accessKey == "" || secretKey == "" || bucketName == "" {
		return errors.New(fmt.Sprintf("Environment variables are not loaded correctly"))
	}

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String("us-west-2"),
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
	})
	if err != nil {
		return errors.New(fmt.Sprintf("Error creating session: %s", err.Error()))
	}

	client := s3.New(sess)

	f, err := os.Open(file.Filename)
	defer f.Close()
	if err != nil {
		return errors.New(fmt.Sprintf("Error opening file: %s", err.Error()))
	}

	// Read the contents of the file into a buffer
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, f); err != nil {
		return errors.New(fmt.Sprintf("Error reading file: %s", err.Error()))
	}

	destinationKey := "cloudbuddy/" + file.Filename

	// This uploads the contents of the buffer to S3
	_, err = client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(destinationKey),
		Body:   bytes.NewReader(buf.Bytes()),
	})

	if err != nil {
		return errors.New(fmt.Sprintf("Error uploading file: %s", err.Error()))
	}

	return nil
}
