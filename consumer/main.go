package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/Shopify/sarama"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// HTTP metrics
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"endpoint", "code"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"endpoint"},
	)

	// Kafka metrics
	kafkaMessagesConsumedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "kafka_messages_consumed_total",
			Help: "Total number of Kafka messages consumed",
		},
	)

	kafkaConsumeDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "kafka_consume_duration_seconds",
			Help:    "Duration of Kafka message consumption in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)

	// Kafka metrics
	prometheus.MustRegister(kafkaMessagesConsumedTotal)
	prometheus.MustRegister(kafkaConsumeDuration)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		httpRequestDuration.WithLabelValues("/").Observe(duration)
		httpRequestsTotal.WithLabelValues("/", "200").Inc()
	}()
	w.Write([]byte("Hello, World!"))
}

func wtfHandler(w http.ResponseWriter, r *http.Request) {
	httpRequestDuration.WithLabelValues("/wtf").Observe(time.Since(time.Now()).Seconds())

	if rand.Float32() < 0.5 { // 50% chance of error
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Something went wrong!"))
		httpRequestsTotal.WithLabelValues("/wtf", "500").Inc()
	} else {
		w.Write([]byte("All good!"))
		httpRequestsTotal.WithLabelValues("/wtf", "200").Inc()
	}
}

func messagesHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		httpRequestDuration.WithLabelValues("/messages").Observe(duration)
		httpRequestsTotal.WithLabelValues("/messages", "200").Inc()
	}()

	messages := getMessages()
	json.NewEncoder(w).Encode(messages)
}

func main() {
	start := time.Now()
	rand.Seed(time.Now().UnixNano())
	brokers := []string{"kafka:9092"} // Modify this as per your Kafka setup
	topic := "random_topic"           // Modify this to your topic name

	partitionConsumer, err := setupKafkaConsumer(brokers, topic)
	if err != nil {
		log.Fatalf("Error setting up Kafka consumer: %v", err)
	}

	go pollMessages(partitionConsumer)

	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/wtf", wtfHandler)
	// http.HandleFunc("/messages", messagesHandler)
	http.HandleFunc("/messages", func(w http.ResponseWriter, r *http.Request) {
		httpRequestDuration.WithLabelValues("/messages").Observe(time.Since(start).Seconds())
		httpRequestsTotal.WithLabelValues("/messages", "200").Inc()
		// Directly serve from the global `messages` slice
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messages)
	})

	http.Handle("/metrics", promhttp.Handler())

	fmt.Println("Server is listening on :8080")
	http.ListenAndServe(":8080", nil)
}

func getMessages() []string {
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	broker := "kafka:9092"
	master, err := sarama.NewConsumer([]string{broker}, config)
	if err != nil {
		panic(err)
	}
	defer master.Close()

	consumer, err := master.ConsumePartition("random_topic", 0, sarama.OffsetOldest)
	if err != nil {
		panic(err)
	}

	messages := []string{}
	for i := 0; i < 30; i++ { // fetch last 10 messages
		select {
		case msg := <-consumer.Messages():
			messages = append(messages, string(msg.Value))
		case <-time.After(30 * time.Second):
			break
		}
	}
	for _, message := range messages {
		fmt.Println(string(message))
		kafkaMessagesConsumedTotal.Inc()
	}

	return messages
}

var messages = make([]string, 0) // Globally accessible slice

func waitForKafka(brokers []string, config *sarama.Config, timeout time.Duration) error {
	endTime := time.Now().Add(timeout)
	for time.Now().Before(endTime) {
		_, err := sarama.NewClient(brokers, config)
		if err == nil {
			return nil
		}
		log.Println("Failed to connect to Kafka, retrying in 5 seconds...")
		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("failed to connect to Kafka after %v", timeout)
}

func setupKafkaConsumer(brokers []string, topic string) (sarama.PartitionConsumer, error) {
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true

	// Wait for Kafka to be available
	timeout := 2 * time.Minute // Adjust this duration as needed
	if err := waitForKafka(brokers, config, timeout); err != nil {
		log.Fatalf("Could not connect to Kafka: %v", err)
	}

	client, err := sarama.NewClient(brokers, config)
	if err != nil {
		log.Fatalf("Error creating Kafka client: %s", err)
	}

	consumer, err := sarama.NewConsumerFromClient(client)
	if err != nil {
		log.Fatalf("Error creating Kafka consumer: %s", err)
	}

	partitionConsumer, err := consumer.ConsumePartition(topic, 0, sarama.OffsetNewest)
	if err != nil {
		log.Fatalf("Error creating Kafka partition consumer: %s", err)
	}

	return partitionConsumer, nil
}

func pollMessages(partitionConsumer sarama.PartitionConsumer) {
	for {
		select {
		case msg := <-partitionConsumer.Messages():
			start := time.Now() // Start timing here
			messages = append(messages, string(msg.Value))
			// Retention logic - keep only last 30 messages
			if len(messages) > 30 {
				messages = messages[1:]
			}
			log.Println("Received message from Kafka:", string(msg.Value))
			kafkaMessagesConsumedTotal.Inc() // Metric counter increase
			duration := time.Since(start).Seconds()
			kafkaConsumeDuration.Observe(duration) // Log the duration here
		case err := <-partitionConsumer.Errors():
			log.Println("Error while consuming from Kafka:", err)
		}
	}
}
