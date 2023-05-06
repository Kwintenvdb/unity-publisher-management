package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron"
)

type schedulingJob struct {
	Publisher     string `json:"publisher"`
	KharmaSession string `json:"kharmaSession"`
	KharmaToken   string `json:"kharmaToken"`
	JWT           string `json:"jwt"`
}

func main() {
	scheduledJobs := map[string]schedulingJob{}

	scheduler := gocron.NewScheduler(time.UTC)
	scheduler.Every(30).Seconds().Do(func() {
		println("Scheduled task running...")
		println("Nr of scheduled jobs: ", len(scheduledJobs))

		for _, job := range scheduledJobs {
			fetchData(job)
		}
	})
	
	// TODO for now the scheduler is invoked directly via HTTP.
	// In the future we will extract this to work via a message queue.
	r := gin.Default()

	r.POST("/schedule", func(c *gin.Context) {
		var job schedulingJob
		err := c.BindJSON(&job)
		if err != nil {
			c.String(400, "Failed to parse job")
			return
		}

		scheduledJobs[job.Publisher] = job
		if len(scheduledJobs) == 1 {
			// Won't start if already started
			scheduler.StartAsync()
			scheduler.RunAll()
		}

		c.String(200, "Job scheduled")
	})

	r.Run(":8083")
}

type MonthData struct {
	Value string `json:"value"`
	Name  string `json:"name"`
}

type SalesData struct {
	PackageName string `json:"package_name"`
	Price       string `json:"price"`
	Sales       int    `json:"sales"`
	Gross       string `json:"gross"`
	LastSale    string `json:"last_sale"`
}

func fetchData(job schedulingJob) {
	// Fetch months
	println("Fetching months...")

	client := http.Client{}

	req := createRequest(fmt.Sprintf("http://localhost:8081/api/months/%s", job.Publisher), job)
	res, err := client.Do(req)
	if err != nil {
		println("Failed to fetch months")
		return
	}

	var months []MonthData
	err = json.NewDecoder(res.Body).Decode(&months)
	if err != nil {
		println("Failed to parse months")
		return
	}

	// Fetch sales
	println("Fetching sales...")
	for _, month := range months {
		go fetchAndCacheSales(job, month, &client)
	}
}

// TODO extract some duplicate code
func fetchAndCacheSales(job schedulingJob, month MonthData, client *http.Client) {
	req := createRequest(fmt.Sprintf("http://localhost:8081/api/sales/%s/%s", job.Publisher, month.Value), job)
	res, err := client.Do(req)
	if err != nil {
		println("Failed to fetch sales")
		return
	}

	var sales []SalesData
	err = json.NewDecoder(res.Body).Decode(&sales)
	if err != nil {
		println("Failed to parse sales")
		return
	}

	println("Sales for month ", month.Value, ": ", sales)

	// Cache the sales
	println("Caching sales...")

	cacheUrl := fmt.Sprintf("http://localhost:8082/sales/%s/%s", job.Publisher, month.Value)

	salesData, _ := json.Marshal(sales)
	_, err = http.Post(cacheUrl, "application/json", bytes.NewReader(salesData))
	if err != nil {
		println("Failed to cache sales")
	}
}

func createRequest(url string, job schedulingJob) *http.Request {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.AddCookie(&http.Cookie{
		Name:  "kharma_token",
		Value: job.KharmaToken,
	})
	req.AddCookie(&http.Cookie{
		Name:  "kharma_session",
		Value: job.KharmaSession,
	})
	req.AddCookie(&http.Cookie{
		Name:  "jwt",
		Value: job.JWT,
	})
	return req
}
