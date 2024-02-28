package main

import (
	"beautybargains/models"
	"beautybargains/repositories"
	"beautybargains/services"
	"database/sql"
	"log"
	"regexp"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// get all posts
	db, err := sql.Open("sqlite3", "data")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	service := services.NewService(db)

	promotions, err := service.GetBannerPromotions(services.GetBannerPromotionsParams{})
	if err != nil {
		log.Fatal(err)
	}

	hashtags, hdb, err := repositories.DefaultHashtagRepoConnection()
	if err != nil {
		log.Fatal(err)
	}
	defer hdb.Close()

	// Define a regular expression pattern for hashtags
	pattern := regexp.MustCompile(`#(\w+)`)

	for _, p := range promotions {
		// Find all matches in the post
		matches := pattern.FindAllStringSubmatch(p.Description, -1)

		// Extract hashtags from the matches
		for _, match := range matches {
			for _, phrase := range match {
				exists, err := hashtags.DoesHashtagExist(phrase)
				if err != nil {
					log.Fatal(err)
				}

				if exists {
					// add the relationship to the database
				} else {
					_, err := hashtags.Insert(&models.Hashtag{Phrase: phrase})
					if err != nil {
						log.Fatal(err)
					}
				}
			}

		}
	}
}

/* Clean up
personas, pdb, err := repositories.DefaultPersonaRepoConnection()
	if err != nil {
		log.Fatal(err)
	}
	defer pdb.Close()

	personaPhotos, phdb, err := repositories.DefaultProfilePhotoRepoConnection()
	if err != nil {
		log.Fatal(err)
	}
	defer phdb.Close()

	allPersonas, err := personas.GetAll()
	if err != nil {
		log.Fatal(err)
	}

	for _, persona := range allPersonas {
		images, err := personaPhotos.GetRandomModelImages(persona.ID, 1)
		if err != nil {
			log.Fatal(err)
		}
		if len(images) < 1 {
			log.Printf("No images for %s. Going to have to remove.", persona.Name)
			err := personas.Delete(persona.ID)
			if err != nil {
				log.Printf("Could not delete %s. %v", persona.Name, err)
			}
			continue
		}
		img := images[0]
		persona.ProfilePhoto = img.URL
		_, err = personas.Update(*persona)
		if err != nil {
			log.Fatal(err)
		}
	}


*/
