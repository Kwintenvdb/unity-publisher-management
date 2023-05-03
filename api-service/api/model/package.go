package model

type PackageData struct {
	Id            string `json:"id"`
	Name          string `json:"name"`
	Url           string `json:"short_url"`
	AverageRating string `json:"average_rating"`
	NumRatings    string `json:"count_ratings"`
}
