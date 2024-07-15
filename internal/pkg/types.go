package pkg

import "time"

type Image struct {
	UUID      string    `clover:"_id" json:"uuid"`
	Title     string    `clover:"title" json:"title"`
	Url       string    `clover:"url" json:"image_url"`
	Likes     int64     `clover:"likes" json:"likes"`
	UserId    string    `clover:"user_id" json:"user_id"`
	CreatedAt time.Time `clover:"created_at" json:"created_at"`
}

type User struct {
	UUID       string    `clover:"_id" json:"uuid"`
	Username   string    `clover:"username" json:"username"`
	Fullname   string    `clover:"fullname" json:"fullname"`
	Passphrase string    `clover:"passphrase" json:"-"`
	Images     []Image   `clover:"images" json:"-"`
	CreatedAt  time.Time `clover:"created_at" json:"created_at"`
}
