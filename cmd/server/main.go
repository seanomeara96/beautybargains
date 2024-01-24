package main

import (
	"beautybargains/handlers"
	"beautybargains/models"
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
		log.Println("port is required via -port flag")
		return
	}

	if mode == "" {
		log.Println("no mode was supplied, startig server in dev mode")
		mode = models.Dev
	}

	db, err := sql.Open("sqlite3", "data")
	if err != nil {
		panic(err)
	}

	tmpl, err := template.New("web").Funcs(funcMap).ParseGlob("templates/**/*.tmpl")
	if err != nil {
		panic(err)
	}

	service := services.NewService(db)
	handler := handlers.NewHandler(service, tmpl)
	router := routers.NewRouter(mode, handler)
	log.Println("listening on " + port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
