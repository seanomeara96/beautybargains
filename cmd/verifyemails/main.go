package main

import (
	"beautybargains/internal/services/mailingsvc"
	"database/sql"
	"log"
)

func main() {

	db, err := sql.Open("sqlite3", "data")
	if err != nil {
		log.Printf("Error: Could not connect to database %v", err)
		return
	}
	defer db.Close()

	service := mailingsvc.New(db)

	emails, err := service.GetUnverifiedEmails()
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	for _, email := range emails {
		token, err := service.GenerateToken(10)
		if err != nil {
			log.Printf("Error: %v", err)
			return
		}

		err = service.AddVerficationToken(email, token)
		if err != nil {
			log.Printf("Error: %v")
			return
		}

		err = service.SendVerificationEmail(email, token)
		if err != nil {
			log.Printf("Error: %v")
			return
		}

	}

}
