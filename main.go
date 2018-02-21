package main

import (
	"crypto/tls"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	gomail "gopkg.in/gomail.v2"

	"github.com/PuerkitoBio/goquery"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

var city1, city2, mail_host, mail_from, mail_secret, mail_to, database string
var mail_port int

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	city1 = os.Getenv("CITY1")
	city2 = os.Getenv("CITY2")
	mail_host = os.Getenv("MAIL_HOST")
	mail_port, _ = strconv.Atoi(os.Getenv("MAIL_PORT"))
	mail_from = os.Getenv("MAIL_FROM")
	mail_secret = os.Getenv("MAIL_SECRET")
	mail_to = os.Getenv("MAIL_TO")
	database = "checknews.db"
	ScripeSite()
}

func ScripeSite() {

	db, err := sql.Open("sqlite3", "./"+database)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Query("SELECT datetime FROM userinfo WHERE 1=0")
	if err != nil {
		// Must create table
		db.Exec(`CREATE TABLE news (datetime CHAR (50) PRIMARY KEY NOT NULL DEFAULT (''), title TEXT DEFAULT (''), body TEXT NOT NULL DEFAULT (''), sent NUMERIC (1) NOT NULL DEFAULT (0))`)
	}

	doc, err := goquery.NewDocument("http://www.021.rs/Novi-Sad/Servisne-informacije/6")
	if err != nil {
		log.Fatal(err)
	}

	// Find the review items
	doc.Find(".article_wrapper").Each(func(i int, s *goquery.Selection) {
		// For each item found, get the band and title
		date := s.Find(".article-info").Text()
		t := strings.Replace(date, " ", "", -1)
		t = strings.Replace(t, "\n", "", -1)
		t = strings.Replace(t, "\r", "", -1)
		title := s.Find(".aTitle a span").Text()
		text := s.Find(".storyIntro").Text()
		if (strings.Contains(title, "Isključenja") || strings.Contains(title, "isključenja") || strings.Contains(title, "struje") || strings.Contains(title, "Struje")) &&
			(strings.Contains(text, city1) || strings.Contains(text, city2)) {
			fmt.Printf("%d: %s - %s - %s\n", i, t, title, text)

			tx, err := db.Begin()
			if err != nil {
				log.Fatal(err)
			}
			stmt, err := tx.Prepare("insert into news(datetime, title, body, sent) values(?, ?, ?, ?)")
			if err != nil {
				fmt.Printf("%s\n", err)
				//log.Fatal(err)
			}
			defer stmt.Close()
			_, err = stmt.Exec(strings.TrimSpace(t), title, text, 0)
			if err != nil {
				fmt.Printf("%s\n", err)
				//log.Fatal(err)
			}
			tx.Commit()

		}
	})

	fmt.Println("OK")

	rows, err := db.Query("SELECT datetime, title, body FROM news WHERE sent = ?", 0)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var datetime string
		var title string
		var body string
		err = rows.Scan(&datetime, &title, &body)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(datetime, title, body)
		m := gomail.NewMessage()
		fmt.Println(mail_from, mail_to, mail_host, mail_port, mail_secret)
		m.SetHeader("From", mail_from)
		m.SetHeader("To", mail_to)
		m.SetHeader("Subject", datetime+" "+title)
		m.SetBody("text/html", body)
		d := gomail.NewDialer(mail_host, mail_port, mail_from, mail_secret)
		d.TLSConfig = &tls.Config{InsecureSkipVerify: true}

		if errm := d.DialAndSend(m); errm != nil {
			fmt.Println(errm)
			fmt.Println(`E-Mail not sent!`)
		} else {
			fmt.Println(`E-Mail sent!`)

			tx, err := db.Begin()
			if err != nil {
				log.Fatal(err)
			}
			stmt, err := tx.Prepare("UPDATE news SET sent=? WHERE datetime=?")
			if err != nil {
				fmt.Printf("%s\n", err)
				//log.Fatal(err)
			}
			defer stmt.Close()
			datetime = strings.TrimSpace(datetime)
			bb := datetime
			fmt.Println(datetime + "%")
			_, err = stmt.Exec(1, bb)
			if err != nil {
				fmt.Printf("%s\n", err)
				//log.Fatal(err)
			}
			tx.Commit()
		}
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
}
