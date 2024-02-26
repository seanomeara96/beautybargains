package main

import (
	"beautybargains/handlers"
	"beautybargains/models"
	"beautybargains/repositories"
	"beautybargains/routers"
	"beautybargains/services"
	"database/sql"
	"flag"
	"html/template"
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	_port := flag.String("port", "", "http port")
	_mode := flag.String("mode", "", "deployment mode")

	flag.Parse()

	port := *_port
	mode := models.Mode(*_mode)

	if port == "" {
		log.Println("Port is required via -port flag.")
		return
	}

	if mode == "" {
		log.Println("No mode was supplied, starting server in development mode.")
		mode = models.Dev
	}

	db, err := sql.Open("sqlite3", "data")
	if err != nil {
		log.Printf("Issue connecting to database. %v", err)
		return
	}
	defer db.Close()

	tmpl, err := template.New("web").Funcs(funcMap).ParseGlob("templates/**/*.tmpl")
	if err != nil {
		log.Printf("Error parsing templates. %v", err)
		return
	}

	personaRepo, personaDB, err := repositories.DefaultPersonaRepoConnection()
	if err != nil {
		log.Fatal(err)
	}
	defer personaDB.Close()
	service := services.NewService(db)
	handler := handlers.NewHandler(service, tmpl, services.NewPersonaService(personaRepo))
	router := routers.NewRouter(mode, handler)

	log.Println("Server listening on http://localhost:" + port)
	if err = http.ListenAndServe(":"+port, router); err != nil {
		log.Printf("Error during listen and serve. %v", err)
	}
}
