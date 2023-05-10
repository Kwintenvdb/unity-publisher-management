package auth

import (
	"context"
	"encoding/json"
	"fmt"

	kafka "github.com/segmentio/kafka-go"
)

type schedulingJob struct {
	Publisher     string `json:"publisher"`
	KharmaSession string `json:"kharmaSession"`
	KharmaToken   string `json:"kharmaToken"`
	JWT           string `json:"jwt"`
}

func SendUserAuthenticatedMessage(publisher, session, token, jwt string) {
	w := kafka.Writer{
		Addr:     kafka.TCP("localhost:61162"),
		Topic:    "user.authentications",
		Balancer: &kafka.LeastBytes{},
	}

	job := schedulingJob{
		Publisher:     publisher,
		KharmaSession: session,
		KharmaToken:   token,
		JWT:           jwt,
	}

	messageKey := fmt.Sprintf("user.auth.%s", publisher)
	message, err := json.Marshal(job)
	if err != nil {
		panic(err)
	}

	err = w.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte(messageKey),
			Value: message,
		},
	)

	if err != nil {
		panic(err)
	}
}
