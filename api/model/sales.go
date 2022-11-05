package model

import (
	"strconv"
)

type RawSalesData struct {
	AaData [][]string `json:"aaData"`
}

type SalesData struct {
	PackageName string `json:"package_name"`
	Price       string `json:"price"`
	Sales       int    `json:"sales"`
	Gross       string `json:"gross"`
	LastSale    string `json:"last_sale"`
}

func SalesFromRaw(rawSalesData RawSalesData) []SalesData {
	var sales []SalesData
	for _, data := range rawSalesData.AaData {
		numSales, _ := strconv.Atoi(data[2])
		s := SalesData{
			PackageName: data[0],
			Price:       data[1],
			Sales:       numSales,
			Gross:       data[5],
			LastSale:    data[7],
		}
		sales = append(sales, s)
	}
	return sales
}
