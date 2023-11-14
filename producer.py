from flask import Flask, jsonify
from threading import Thread
from confluent_kafka import Producer, KafkaException
from prometheus_client import start_http_server, Counter
from prometheus_kafka_producer.metrics_manager import ProducerMetricsManager
from datetime import datetime
import sys
import time
import json
import string
import random

# Metrics collector
metric_manager = ProducerMetricsManager()

# Flask app
app = Flask(__name__)

# Counter for produced messages
produced_messages_counter = Counter('produced_messages', 'Total produced messages')

# Producer configuration
BROKERS = "kafka:9092"
TOPIC = 'random_topic'
CONF = {
    'bootstrap.servers': BROKERS,
    'stats_cb': metric_manager.send,
    'statistics.interval.ms': 1000
}

# Kafka producer
producer = Producer(**CONF)

def delivery_report(err, msg):
    """ Called once for each message produced to indicate delivery result. """
    if err is not None:
        sys.stderr.write(f"{datetime.now().strftime('%Y/%m/%d %H:%M:%S')} Message delivery failed: {err}\n")
    else:
        sys.stderr.write(f"{datetime.now().strftime('%Y/%m/%d %H:%M:%S')} Message delivered to {msg.topic()} into partition [{msg.partition()}]\n")

def wait_for_kafka():
    """ Wait for Kafka to become available. """
    while True:
        try:
            # Try to get metadata to see if Kafka is up
            producer.list_topics(timeout=5)
            break
        except KafkaException as e:
            if "Failed to resolve" in str(e):
                sys.stderr.write(f"Unable to resolve {BROKERS}. Retrying...\n")
            else:
                sys.stderr.write(f"Error: {e}\n")
            time.sleep(10)

# Produce a message and update the counter
def produce_message():
    wait_for_kafka()  # Ensure Kafka is up before starting message production
    while True:
        message = {
                'name': ''.join(random.choice(string.ascii_uppercase + string.digits) for _ in range(10)),
                'date': datetime.now().strftime('%Y-%m-%d %H:%M:%S')
            }
        producer.produce(TOPIC, key='key', value=json.dumps(message), callback=delivery_report)
        producer.flush()
        produced_messages_counter.inc()  # Increment Prometheus counter
        time.sleep(10)  # Produce a message every 10 seconds

# Endpoint to get the message count
@app.route('/')
def get_message_count():
    return jsonify(message_counter=produced_messages_counter._value.get())

# Function to run the Flask server on a different port
def run_flask():
    app.run(host='0.0.0.0', port=5001, debug=False)

# Function to start the metrics server
def run_metrics_server():
    start_http_server(8090)

if __name__ == "__main__":
    # Start the Flask server in a separate thread
    flask_thread = Thread(target=run_flask, daemon=True)
    flask_thread.start()

    # Start the Prometheus metrics server
    metrics_thread = Thread(target=run_metrics_server, daemon=True)
    metrics_thread.start()

    # Start producing messages
    produce_message_thread = Thread(target=produce_message, daemon=True)
    produce_message_thread.start()

    # Keep the main thread alive
    flask_thread.join()
    metrics_thread.join()
    produce_message_thread.join()
