package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jasonlvhit/gocron"
)

type schedulingJob struct {
	Publisher     string `json:"publisher"`
	KharmaSession string `json:"kharmaSession"`
	KharmaToken   string `json:"kharmaToken"`
	JWT           string `json:"jwt"`
}

func main() {
	scheduledJobs := map[string]schedulingJob{}

	gocron.Every(10).Seconds().Do(func() {
		println("Scheduled task running...")
		println("Nr of scheduled jobs: ", len(scheduledJobs))

		for _, job := range scheduledJobs {
			fetchData(job)
		}
	})

	go func() {
		<-gocron.Start()
	}()

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
		c.String(200, "Job scheduled")
	})

	r.Run(":8083")
}

type MonthData struct {
	Value string `json:"value"`
	Name  string `json:"name"`
}

func fetchData(job schedulingJob) {
	// Fetch months
	println("Fetching months...")

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:8081/api/months/%s", job.Publisher), nil)
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

	client := http.Client{}
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
}
