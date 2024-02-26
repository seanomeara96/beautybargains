package repositories

import (
	"beautybargains/models"
	"database/sql"
)

type ProfilePhotoRepo struct {
	db *sql.DB
}

func NewProfilePhotoRepo(db *sql.DB) *ProfilePhotoRepo {
	return &ProfilePhotoRepo{db}
}

func DefaultProfilePhotoRepoConnection() (*ProfilePhotoRepo, *sql.DB, error) {
	db, err := sql.Open("sqlite3", "images.db")
	if err != nil {
		return nil, nil, err
	}
	return NewProfilePhotoRepo(db), db, nil
}

func (s *ProfilePhotoRepo) GetModelNames(randomOrder bool) ([]string, error) {
	names := []string{}
	q := "SELECT DISTINCT name FROM images"
	if randomOrder {
		q += " ORDER BY RANDOM()"
	}

	rows, err := s.db.Query(q)
	if err != nil {
		return names, err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		err := rows.Scan(&name)
		if err != nil {
			return names, err
		}
		names = append(names, name)
	}

	return names, nil
}

func (s *ProfilePhotoRepo) GetRandomImages(limit int) ([]models.ProfilePhoto, error) {
	images := []models.ProfilePhoto{}
	q := "SELECT id, file_path as url, name FROM images ORDER BY RANDOM() LIMIT ?"
	rows, err := s.db.Query(q, limit)
	if err != nil {
		return images, err
	}
	defer rows.Close()

	for rows.Next() {
		image := models.ProfilePhoto{}
		err = rows.Scan(&image.ID, &image.URL, &image.Name)
		if err != nil {
			return images, err
		}
		images = append(images, image)
	}

	return images, nil
}

func (s *ProfilePhotoRepo) GetRandomModelImages(modelID int, limit int) ([]models.ProfilePhoto, error) {
	images := []models.ProfilePhoto{}
	q := "SELECT id, file_path as url, name FROM images WHERE model_id = ? ORDER BY RANDOM() LIMIT ?"
	rows, err := s.db.Query(q, modelID, limit)
	if err != nil {
		return images, err
	}
	defer rows.Close()

	for rows.Next() {
		image := models.ProfilePhoto{}
		err = rows.Scan(&image.ID, &image.URL, &image.Name)
		if err != nil {
			return images, err
		}
		images = append(images, image)
	}

	return images, nil
}

func (s *ProfilePhotoRepo) DeleteImageByID(id int) error {
	q := "DELETE FROM images WHERE  id = ?"
	_, err := s.db.Exec(q, id)
	return err
}

func (s *ProfilePhotoRepo) GetAllImages() ([]models.ProfilePhoto, error) {
	images := []models.ProfilePhoto{}
	q := "SELECT id, file_path as url, name FROM images"
	rows, err := s.db.Query(q)
	if err != nil {
		return images, err
	}
	defer rows.Close()

	for rows.Next() {
		image := models.ProfilePhoto{}
		err := rows.Scan(&image.ID, &image.URL, &image.Name)
		if err != nil {
			return images, err
		}
		images = append(images, image)
	}

	return images, nil
}

func (s *ProfilePhotoRepo) GetAllModelImages(name string) ([]models.ProfilePhoto, error) {
	images := []models.ProfilePhoto{}
	q := "SELECT id, file_path as url, name FROM images WHERE name = ?"
	rows, err := s.db.Query(q, name)
	if err != nil {
		return images, err
	}
	defer rows.Close()

	for rows.Next() {
		image := models.ProfilePhoto{}
		err := rows.Scan(&image.ID, &image.URL, &image.Name)
		if err != nil {
			return images, err
		}
		images = append(images, image)
	}

	return images, nil
}

func (s *ProfilePhotoRepo) GetImageByID(id int) (models.ProfilePhoto, error) {
	image := models.ProfilePhoto{}
	q := "SELECT id, file_path as url, name FROM images WHERE id = ?"
	err := s.db.QueryRow(q, id).Scan(&image.ID, &image.URL, &image.Name)
	if err != nil {
		return image, err
	}
	return image, nil
}

func (s *ProfilePhotoRepo) GetImageFilePathByID(id int) (string, error) {
	var filepath string
	q := "SELECT file_path FROM images WHERE id = ?"
	err := s.db.QueryRow(q, id).Scan(&filepath)
	if err != nil {
		return filepath, err
	}
	return filepath, nil
}
