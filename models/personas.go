package models

type Persona struct {
	ID           int
	Name         string
	Description  string
	ProfilePhoto string
}

type ProfilePhoto struct {
	ID      int
	URL     string
	Name    string
	ModelID int
}
