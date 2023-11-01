# producer.py

import time
import json
import sys
from confluent_kafka import Producer, KafkaException
from datetime import datetime

BROKERS = "kafka:9092"
TOPIC = 'random_topic'
CONF = {
    'bootstrap.servers': BROKERS,
}
RETRY_INTERVAL = 10
WAIT_TIMEOUT = 120


def wait_for_kafka(brokers=BROKERS, timeout=WAIT_TIMEOUT):
    """Wait for Kafka to be available."""
    end_time = time.time() + timeout

    while time.time() < end_time:
        temporary_producer = Producer({'bootstrap.servers': brokers})

        try:
            # Attempt to retrieve metadata to see if Kafka is available
            temporary_producer.list_topics(timeout=5)
            return
        except KafkaException as e:
            if "Failed to resolve" in str(e):
                print(f"Unable to resolve {brokers}. Retrying...", file=sys.stderr)
            else:
                # Handle other Kafka exceptions if needed
                print(f"Error: {e}", file=sys.stderr)
            time.sleep(RETRY_INTERVAL)

    raise TimeoutError(f"Unable to connect to Kafka after {timeout} seconds")


def delivery_report(err, msg):
    """ Called once for each message produced to indicate delivery result.
        Triggered by poll() or flush(). """
    if err is not None:
        print('Message delivery failed: {}'.format(err), file=sys.stderr)
    else:
        print('Message delivered to {} into partition [{}]'.format(msg.topic(), msg.partition()), file=sys.stderr)


def produce_messages(producer, topic=TOPIC):
    while True:
        try:
            message = {
                'name': 'random_name',
                'date': datetime.now().strftime('%Y-%m-%d %H:%M:%S')
            }
            producer.produce(topic, value=json.dumps(message), callback=delivery_report)
            producer.flush()
        except KafkaException as e:
            if "Failed to resolve" in str(e):
                print(f"Failed to resolve {BROKERS}. Waiting and then retrying...", file=sys.stderr)
            else:
                # Handle other Kafka exceptions if necessary
                print(f"Error while producing: {e}", file=sys.stderr)
            time.sleep(RETRY_INTERVAL)

        time.sleep(RETRY_INTERVAL)


def main():
    # Wait for Kafka
    try:
        wait_for_kafka()
    except TimeoutError as e:
        print(e, file=sys.stderr)
        sys.exit(1)

    producer = Producer(CONF)
    produce_messages(producer)


if __name__ == "__main__":
    main()
