package pricedatarepo

import (
	"beautybargains/internal/models"
	"database/sql"
	"fmt"
)

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{db}
}

// Create a new price data entry and insert it into the PriceData table
func (s *Repository) CreatePriceData(priceData models.PriceData) error {

	q := `INSERT INTO PriceData (
		ProductID, 
		Price, 
		Currency, 
		Timestamp, 
		Gtin12, 
		Gtin13, 
		Gtin14, 
		SKU, 
		Name, 
		Image
	) VALUES 
		(?, ?, ?, ?, ?, ?, ? ,? ,? ,?)`

	if _, err := s.db.Exec(q,
		priceData.ProductID,
		priceData.Price,
		priceData.Currency,
		priceData.Timestamp,
		priceData.Gtin12,
		priceData.Gtin13,
		priceData.Gtin14,
		priceData.SKU,
		priceData.Name,
		priceData.Image,
	); err != nil {
		return err
	}
	return nil
}

func (r *Repository) CountByProductID(id int) (int, error) {
	q := `SELECT
		count(PriceID)
	FROM
		PriceData
	WHERE
		ProductID = ?`

	var count int
	if err := r.db.QueryRow(q, id).Scan(&count); err != nil {
		return -1, fmt.Errorf("Error querying count by product id. %w", err)
	}
	return count, nil
}

// Retrieve price data by its ID from the PriceData table
func (s *Repository) GetPriceData(productID, priceID int) (models.PriceData, error) {
	q := `SELECT 
		PriceID, 
		ProductID, 
		Price, 
		Currency, 
		Timestamp, 
		Gtin12, 
		Gtin13, 
		Gtin14, 
		SKU, 
		Name, 
		Image 
	FROM 
		PriceData 
	WHERE 
		PriceID = ? 
	AND 
		ProductID = ?`

	var priceData models.PriceData

	if err := s.db.QueryRow(q, priceID, productID).
		Scan(
			&priceData.PriceID,
			&priceData.ProductID,
			&priceData.Price,
			&priceData.Currency,
			&priceData.Timestamp,
			&priceData.Gtin12,
			&priceData.Gtin13,
			&priceData.Gtin14,
			&priceData.SKU,
			&priceData.Name,
			&priceData.Image); err != nil {
		return models.PriceData{}, err
	}

	return priceData, nil
}
func (s *Repository) GetLatestPrice(productID int) (models.PriceData, error) {
	q := `SELECT 
		PriceID, 
		ProductID, 
		Price, 
		Currency, 
		Timestamp,
		MAX(Timestamp) as Latest, 
		Gtin12, 
		Gtin13, 
		Gtin14, 
		SKU, 
		Name, 
		Image 
	FROM 
		PriceData 
	WHERE
		ProductID = ?`

	var priceData models.PriceData
	var temp string
	if err := s.db.QueryRow(q, productID).
		Scan(
			&priceData.PriceID,
			&priceData.ProductID,
			&priceData.Price,
			&priceData.Currency,
			&priceData.Timestamp,
			&temp,
			&priceData.Gtin12,
			&priceData.Gtin13,
			&priceData.Gtin14,
			&priceData.SKU,
			&priceData.Name,
			&priceData.Image); err != nil {
		return models.PriceData{}, err
	}

	return priceData, nil
}

// Retrieve price data by its ID from the PriceData table
func (s *Repository) GetPriceDataByID(priceID int) (models.PriceData, error) {
	var priceData models.PriceData
	err := s.db.QueryRow("SELECT PriceID, ProductID, Price, Currency, Timestamp, Gtin12, Gtin13, Gtin14, SKU, Name, Image FROM PriceData WHERE PriceID = ?", priceID).
		Scan(&priceData.PriceID, &priceData.ProductID, &priceData.Price, &priceData.Currency, &priceData.Timestamp, &priceData.Gtin12, &priceData.Gtin13, &priceData.Gtin14, &priceData.SKU, &priceData.Name, &priceData.Image)
	if err != nil {
		return models.PriceData{}, err
	}
	return priceData, nil
}
func (s *Repository) GetByProductID(productID int) ([]models.PriceData, error) {
	rows, err := s.db.Query("SELECT PriceID, ProductID, Price, Currency, Timestamp, Gtin12, Gtin13, Gtin14, SKU, Name, Image FROM PriceData WHERE ProductID = ?", productID)
	if err != nil {
		return []models.PriceData{}, err
	}
	prices := []models.PriceData{}
	for rows.Next() {
		var priceData models.PriceData
		err = rows.Scan(&priceData.PriceID, &priceData.ProductID, &priceData.Price, &priceData.Currency, &priceData.Timestamp, &priceData.Gtin12, &priceData.Gtin13, &priceData.Gtin14, &priceData.SKU, &priceData.Name, &priceData.Image)
		if err != nil {
			return []models.PriceData{}, err
		}
		prices = append(prices, priceData)
	}
	return prices, nil
}

// Update an existing price data entry in the PriceData table
func (s *Repository) UpdatePriceData(priceData models.PriceData) error {
	_, err := s.db.Exec("UPDATE PriceData SET ProductID = ?, Price = ?, Currency = ?, Timestamp = ?, Gtin12 = ?, Gtin13 = ?, Gtin14 = ?, SKU = ?, Name = ?, Image = ? WHERE PriceID = ?",
		priceData.ProductID, priceData.Price, priceData.Currency, priceData.Timestamp, priceData.Gtin12, priceData.Gtin13, priceData.Gtin14, priceData.SKU, priceData.Name, priceData.Image, priceData.PriceID)
	if err != nil {
		return err
	}
	return nil
}

// Delete price data entry by its ID from the PriceData table
func (s *Repository) DeletePriceData(priceID int) error {
	_, err := s.db.Exec("DELETE FROM PriceData WHERE PriceID = ?", priceID)
	if err != nil {
		return err
	}
	return nil
}

func (s *Repository) GetPriceDrops(limit, offset int) ([]models.PriceChange, error) {
	q := `SELECT                                                   
    pd.ProductID,
    (SELECT Price FROM PriceData WHERE ProductID = pd.ProductID ORDER BY Timestamp DESC LIMIT 1)  AS CurrentPrice,
    (SELECT Timestamp FROM PriceData WHERE ProductID = pd.ProductID ORDER BY Timestamp DESC LIMIT 1) AS CurrentTimestamp,
    (SELECT Price FROM PriceData WHERE ProductID = pd.ProductID ORDER BY Timestamp DESC LIMIT 1 OFFSET 1) AS PreviousPrice,
    (SELECT Timestamp FROM PriceData WHERE ProductID = pd.ProductID ORDER BY Timestamp DESC LIMIT 1 OFFSET 1) AS PreviousTimestamp
FROM 
    PriceData pd WHERE CurrentPrice < PreviousPrice
GROUP BY
    pd.ProductID
ORDER BY CurrentTimestamp DESC LIMIT ? OFFSET ?`

	rows, err := s.db.Query(q, limit, offset)
	if err != nil {
		return []models.PriceChange{}, err
	}
	defer rows.Close()

	var priceDrops []models.PriceChange

	for rows.Next() {

		var change models.PriceChange
		err := rows.Scan(&change.ProductID, &change.CurrentPrice, &change.CurrentTimeStamp, &change.PreviousPrice, &change.PreviousTimestamp)

		if err != nil {
			return []models.PriceChange{}, err
		}

		priceDrops = append(priceDrops, change)

	}

	return priceDrops, nil
}
