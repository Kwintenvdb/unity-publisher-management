package model

type PackageData struct {
	Id            string  `json:"id"`
	Name          string  `json:"name"`
	Url           string  `json:"short_url"`
	AverageRating float64 `json:"average_rating"`
	NumRatings    int     `json:"count_ratings"`
}
