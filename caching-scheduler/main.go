package main

import (
	"github.com/gin-gonic/gin"
	"github.com/jasonlvhit/gocron"
)

type schedulingJob struct {
	Publisher     string `json:"publisher"`
	KharmaSession string `json:"kharmaSession"`
	KharmaToken   string `json:"kharmaToken"`
}

func main() {
	scheduledJobs := []schedulingJob{}

	gocron.Every(10).Seconds().Do(func() {
		println("Scheduled task running...")
		println("Nr of scheduled jobs: ", len(scheduledJobs))
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

		scheduledJobs = append(scheduledJobs, job)
		c.String(200, "Job scheduled")
	})

	r.Run(":8083")
}
