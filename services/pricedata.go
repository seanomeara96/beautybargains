package services

import (
	"beautybargains/models"
)

// Create a new price data entry and insert it into the PriceData table
func (s *Service) CreatePriceData(priceData models.PriceData) error {
	_, err := s.db.Exec("INSERT INTO PriceData (ProductID, Price, Currency, Timestamp, Gtin12, Gtin13, Gtin14, SKU, Name, Image) VALUES (?, ?, ?, ?, ?, ?, ? ,? ,? ,?)",
		priceData.ProductID, priceData.Price, priceData.Currency, priceData.Timestamp, priceData.Gtin12, priceData.Gtin13, priceData.Gtin14, priceData.SKU, priceData.Name, priceData.Image)
	if err != nil {
		return err
	}
	return nil
}

// Retrieve price data by its ID from the PriceData table
func (s *Service) GetPriceData(productID, priceID int) (models.PriceData, error) {
	var priceData models.PriceData
	err := s.db.QueryRow("SELECT PriceID, ProductID, Price, Currency, Timestamp, Gtin12, Gtin13, Gtin14, SKU, Name, Image FROM PriceData WHERE PriceID = ? AND ProductID = ?", priceID, productID).
		Scan(&priceData.PriceID, &priceData.ProductID, &priceData.Price, &priceData.Currency, &priceData.Timestamp, &priceData.Gtin12, &priceData.Gtin13, &priceData.Gtin14, &priceData.SKU, &priceData.Name, &priceData.Image)
	if err != nil {
		return models.PriceData{}, err
	}
	return priceData, nil
}

// Retrieve price data by its ID from the PriceData table
func (s *Service) GetPriceDataByID(priceID int) (models.PriceData, error) {
	var priceData models.PriceData
	err := s.db.QueryRow("SELECT PriceID, ProductID, Price, Currency, Timestamp, Gtin12, Gtin13, Gtin14, SKU, Name, Image FROM PriceData WHERE PriceID = ?", priceID).
		Scan(&priceData.PriceID, &priceData.ProductID, &priceData.Price, &priceData.Currency, &priceData.Timestamp, &priceData.Gtin12, &priceData.Gtin13, &priceData.Gtin14, &priceData.SKU, &priceData.Name, &priceData.Image)
	if err != nil {
		return models.PriceData{}, err
	}
	return priceData, nil
}
func (s *Service) GetPriceDataByProductID(productID int) ([]models.PriceData, error) {
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
func (s *Service) UpdatePriceData(priceData models.PriceData) error {
	_, err := s.db.Exec("UPDATE PriceData SET ProductID = ?, Price = ?, Currency = ?, Timestamp = ?, Gtin12 = ?, Gtin13 = ?, Gtin14 = ?, SKU = ?, Name = ?, Image = ? WHERE PriceID = ?",
		priceData.ProductID, priceData.Price, priceData.Currency, priceData.Timestamp, priceData.Gtin12, priceData.Gtin13, priceData.Gtin14, priceData.SKU, priceData.Name, priceData.Image, priceData.PriceID)
	if err != nil {
		return err
	}
	return nil
}

// Delete price data entry by its ID from the PriceData table
func (s *Service) DeletePriceData(priceID int) error {
	_, err := s.db.Exec("DELETE FROM PriceData WHERE PriceID = ?", priceID)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) GetPriceDrops(limit, offset int) ([]models.PriceChange, error) {
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
