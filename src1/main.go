package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

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
	r.GET("/export", exportData)

	r.Run(":8080")
}

type Transaction struct {
	ID               string
	INITIATOR_REF_NO string
	SYS_REF_NO       string
	HOST_TRX_DT      string
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
			_, err := db.Exec("INSERT INTO transaction (ID, INITIATOR_REF_NO, SYS_REF_NO, HOST_TRX_DT) VALUES (?, ?, ?, ?)", transaction.ID, transaction.INITIATOR_REF_NO, transaction.SYS_REF_NO, transaction.HOST_TRX_DT)
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
				hostTrxDt := row.Cells[11].String()

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

func exportData(c *gin.Context) {
	rows, err := db.Query("SELECT ID, INITIATOR_REF_NO, SYS_REF_NO, HOST_TRX_DT FROM transaction")
	if err!= nil {
        c.String(http.StatusInternalServerError, "Error querying database: %s", err)
        return
    }
	defer rows.Close()

	var transactions []Transaction
	for rows.Next() {
		var t Transaction
		var hostTrxDtStr string

		err := rows.Scan(&t.ID, &t.INITIATOR_REF_NO, &t.SYS_REF_NO, &hostTrxDtStr)
		if err!= nil {
            c.String(http.StatusInternalServerError, "Error scanning row: %s", err)
            return
        }

		t.HOST_TRX_DT = hostTrxDtStr

		transactions = append(transactions, t)

	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func(){
		defer wg.Done()
        file := xlsx.NewFile()
        sheet, err := file.AddSheet("Sheet1")
        

		header := sheet.AddRow()
		headerCells := []string{"ID", "INITIATOR_REF_NO", "SYS_REF_NO", "HOST_TRX_DT"}
		style := xlsx.NewStyle()
		style.Font.Bold = true
		style.Alignment.Horizontal = "center"

		for _, h := range headerCells {
			cell := header.AddCell()
            cell.Value = h
            cell.SetStyle(style)
        }

		for _, transaction := range transactions{
			row := sheet.AddRow()
			row.AddCell().Value = transaction.ID
			row.AddCell().Value = transaction.INITIATOR_REF_NO
			row.AddCell().Value = transaction.SYS_REF_NO
			row.AddCell().Value = transaction.HOST_TRX_DT

		}
		err = file.Save("../assets/export.xlsx")
		if err!= nil {
            log.Printf("Error saving file: %s", err)
        }
	}()
	go func(){
		defer wg.Done()
        file, err := os.Create("../assets/export.txt")
		if err!= nil {
            log.Fatalf("Error creating file: %s", err)
        }
		defer file.Close()

		file.WriteString("ID,INITIATOR_REF_NO,HOST_TRX_DT\n")

		for _, transaction:= range transactions {
			line := fmt.Sprintf("%s,%s, %s, %s,\n", transaction.ID, transaction.INITIATOR_REF_NO, transaction.SYS_REF_NO, transaction.HOST_TRX_DT)
			file.WriteString(line)
		}
	}()
	wg.Wait()
	c.String(http.StatusOK,"Data Succesfully exported to xlsx and txt")
}