package main

import (
	"fmt"
	"net/http"
	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"os"
	"io"
)

var log = logrus.New()

var rabbit_host = os.Getenv("RABBIT_HOST")
var rabbit_port = os.Getenv("RABBIT_PORT") 
var rabbit_user = os.Getenv("RABBIT_USERNAME")
var rabbit_password = os.Getenv("RABBIT_PASSWORD")

func init() {
	log.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	if os.Getenv("DEBUG") == "true" {
		log.SetLevel(logrus.DebugLevel)
	} else {
		log.SetLevel(logrus.InfoLevel)
	}
}

func main() {

	router := httprouter.New()

	router.POST("/publish/:queue", func(w http.ResponseWriter, r *http.Request, p httprouter.Params){
		submit(w,r,p)
	})

	fmt.Println("Running...")
	log.Fatal(http.ListenAndServe(":8080", router))
}

func submit(writer http.ResponseWriter, request *http.Request, p httprouter.Params) {
	queue := p.ByName("queue")
	
	// Read the request body
	body, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	// Close the body to prevent resource leaks
	defer request.Body.Close()
	
	log.Debugf("Received message: " + string(body))

	conn, err := amqp.Dial("amqp://" + rabbit_user + ":" +rabbit_password + "@" + rabbit_host + ":" + rabbit_port +"/")

	if err != nil {
		log.Errorf("%s: %s", "Failed to connect to RabbitMQ", err)
		http.Error(writer, "Failed to connect to RabbitMQ", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	ch, err := conn.Channel()

	if err != nil {
		log.Errorf("%s: %s", "Failed to open a channel", err)
		http.Error(writer, "Failed to open a channel", http.StatusInternalServerError)
		return
	}

	defer ch.Close()

	// q, err := ch.QueueDeclare(
	// 	queue, // name
	// 	true,   // durable
	// 	false,   // delete when unused
	// 	false,   // exclusive
	// 	false,   // no-wait
	// 	nil,     // arguments
	// )

	// if err != nil {
	// 	log.Warnf("%s: %s", "Failed to declare a queue", err)
	// }

	err = ch.Publish(
		"",     // exchange
		queue, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing {
			ContentType: "application/json",
			Body:        []byte(body),
	})

	if err != nil {
		log.Errorf("%s: %s", "Failed to publish a message", err)
		http.Error(writer, "Failed to publish a queue", http.StatusInternalServerError)
		return
	}

}