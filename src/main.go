package main

import (
	"database/sql"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/tealeg/xlsx"
)

var db *sql.DB

func main() {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Disposition"},
		AllowCredentials: true,
	}))
	var err error
	db, err = ConnectMySQL()
	if err != nil {
		log.Fatalf("Error connecting to database: %s", err)
	}
	// db.Exec("CREATE DATABASE IF NOT EXISTS multimatics")

	// route
	r.GET("/", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "Hello, world")
	})
	r.POST("/upload", UploadFile)

	r.Run(":8080")
}

type Transaction struct {
	ID               string
	INITIATOR_REF_NO string
	SYS_REF_NO       string
	HOST_TRX_DT      time.Time
}

func ConnectMySQL() (*sql.DB, error) {
	username := "fizi"
	password := "fizi123"
	host := "multimatics-mysql"
	port := "3306"
	database := "multimatics"

	db, err := sql.Open("mysql", username+":"+password+"@tcp("+host+":"+port+")/"+database)
	if err != nil {
		log.Fatalf("Error connecting to database: %s", err)
	}

	// testing connection
	err = db.Ping()
	if err != nil {
		log.Fatalf("Error pinging to database: %s", err)
	} else {
		log.Println("Connected to database")
	}

	return db, nil
}

func UploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.String(http.StatusBadRequest, "Bad request: %s", err)
		return
	}

	// save the uploaded file
	if err := c.SaveUploadedFile(file, "../assets/forTraining.xlsx"); err != nil {
		c.String(http.StatusInternalServerError, "Could not save the file: %s", err)
	}

	// process file async
	var wg sync.WaitGroup
	ch := make(chan Transaction)

	// goroutine to insert into the database
	wg.Add(1)
	go func() {
		defer wg.Done()
		for transaction := range ch {
			_, err := db.Exec("INSERT INTO transactions (ID, INITIATOR_REF_NO, SYS_REF_NO, HOST_TRX_DT) VALUES (?, ?, ?, ?)", transaction.ID, transaction.INITIATOR_REF_NO, transaction.SYS_REF_NO, transaction.HOST_TRX_DT)
			if err != nil {
				log.Println("Error inserting record: ", err)
			}
		}
	}()

	// read xlsx file in a separate goroutine
	go func() {
		xlFile, err := xlsx.OpenFile("../assets/forTraining.xlsx")
		if err != nil {
			log.Fatal("Failed to open file: ", err)
		}

		for _, sheet := range xlFile.Sheets {
			for _, row := range sheet.Rows {
				id := row.Cells[1].String()
				initiatorRefNo := row.Cells[3].String()
				sysRefNo := row.Cells[4].String()
				hostTrxDt, _ := row.Cells[8].GetTime(false)

				transaction := Transaction{
					ID:               id,
					INITIATOR_REF_NO: initiatorRefNo,
					SYS_REF_NO:       sysRefNo,
					HOST_TRX_DT:      hostTrxDt,
				}
				ch <- transaction
			}
		}
		close(ch)
	}()

	wg.Wait()
	c.String(http.StatusOK, "File uploaded successfully")
}