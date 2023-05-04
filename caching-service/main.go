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
	salesCache["18076"] = salesByMonth{
		"202303": `
		[
			{
				"package_name": "foobar",
				"price": "$ 9.99",
				"sales": 99,
				"gross": "$ 999.99",
				"last_sale": "2023-10-10"
			}
		]`,
	}

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
