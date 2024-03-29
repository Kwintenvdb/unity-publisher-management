package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-co-op/gocron"
	kafka "github.com/segmentio/kafka-go"
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
	scheduler.Every(5).Minutes().Do(func() {
		println("Scheduled task running...")
		println("Nr of scheduled jobs: ", len(scheduledJobs))

		for _, job := range scheduledJobs {
			fetchData(job)
		}
	})

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{"localhost:61162"},
		Topic:   "user.authentications",
		GroupID: "caching-scheduler",
	})
	defer reader.Close()


	println("Waiting for messages from user.authentications topic...")

	// Poll for new messages from user.authentications topic
	for {
		m, err := reader.ReadMessage(context.Background())
		if err != nil {
			panic(err)
		}

		var job schedulingJob
		err = json.Unmarshal(m.Value, &job)
		if err != nil {
			panic(err)
		}

		println("Received scheduling job for publisher", job.Publisher)

		scheduledJobs[job.Publisher] = job
		if len(scheduledJobs) == 1 {
			// Won't start if already started
			scheduler.StartAsync()
			scheduler.RunAll()
		}
	}
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
	req := createRequest(fmt.Sprintf("http://%s/api/sales/%s/%s", getApiServiceHost(), job.Publisher, month.Value), job)
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

	println("Fetched sales for month", month.Value)

	// Cache the sales
	println("Caching sales...")

	cacheUrl := fmt.Sprintf("http://%s/sales/%s/%s", getCachingServiceHost(), job.Publisher, month.Value)

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

func getApiServiceHost() string {
	if host, found := os.LookupEnv("UPM_API_SERVICE"); found {
		return host
	}
	return "localhost:8081"
}

func getCachingServiceHost() string {
	if host, found := os.LookupEnv("UPM_CACHING_SERVICE"); found {
		return host
	}
	return "localhost:8082"
}
