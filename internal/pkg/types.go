package pkg

import "time"

type Image struct {
	UUID      string    `clover:"_id" json:"uuid"`
	Url       string    `clover:"url" json:"image_url"`
	Likes     int64     `clover:"likes" json:"likes"`
	UserId    string    `clover:"user_id" json:"user_id"`
	CreatedAt time.Time `clover:"created_at" json:"created_at"`
}

type User struct {
	UUID       string    `clover:"_id" json:"uuid"`
	Username   string    `clover:"url" json:"username"`
	Fullname   string    `clover:"fullname" json:"fullname"`
	Passphrase string    `clover:"passphrase" json:"-"`
	CreatedAt  time.Time `clover:"created_at" json:"created_at"`
}
