package main

import (
	"beautybargains/repositories"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func main() {

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

}
