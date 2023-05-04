package main

import (
	"io"

	"github.com/gin-gonic/gin"
)

type salesByMonth = map[string]string
type salesByPublisher = map[string]salesByMonth

func main() {
	r := gin.Default()

	salesCache := make(salesByPublisher)

	r.GET("/sales/:publisher/:month", func(c *gin.Context) {
		publisher := c.Param("publisher")
		month := c.Param("month")

		if salesOfPublisher, ok := salesCache[publisher]; ok {
			if sales, ok := salesOfPublisher[month]; ok {
				c.String(200, sales)
				return
			}
		}

		c.String(404, "Sales not found")
	})

	r.POST("/sales/:publisher/:month", func(c *gin.Context) {
		publisher := c.Param("publisher")
		month := c.Param("month")

		body := c.Request.Body
		data, err := io.ReadAll(body)
		if err != nil {
			c.String(400, "Failed to read sales")
			return
		}

		sales := string(data)

		if salesOfPublisher, ok := salesCache[publisher]; ok {
			salesOfPublisher[month] = sales
		} else {
			salesCache[publisher] = salesByMonth{
				month: sales,
			}
		}

		c.String(200, "Sales cached")
	})

	r.Run(":8082")
}
