package controllers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB

func InitAuth(database *sql.DB) {
	db = database
}

func Register(c *gin.Context) {
	nama := c.PostForm("nama")
	username := c.PostForm("username")
	password := c.PostForm("password")

	file, err := c.FormFile("foto")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gagal membaca file foto, "+err.Error()})
		return
	}

	if nama == "" || username == "" || password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nama, username, dan password wajib diisi, "})
		return
	}

	fotoPath := "./uploads/" + file.Filename
	if err := c.SaveUploadedFile(file, fotoPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan file foto, "+err.Error()})
	}

	hasedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengenkripsi passwrod, "+err.Error()})
		return
	}

	_, err = db.Exec("INSERT INTO users (nama, username, password, foto) VALUES(?,?,?,?)", nama, username, string(hasedPassword), fotoPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan data user, "+err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User berhasil didaftarkan"})

}
