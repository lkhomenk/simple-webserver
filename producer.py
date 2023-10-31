# producer.py

import time
import json
from confluent_kafka import Producer, KafkaError
from datetime import datetime

conf = {
    'bootstrap.servers': 'kafka:9092',
}

def wait_for_kafka(brokers, timeout=120):
    """Wait for Kafka to be available."""
    end_time = time.time() + timeout
    while time.time() < end_time:
        producer = Producer({'bootstrap.servers': brokers})
        try:
            # Try to get metadata to see if Kafka is up
            producer.list_topics(timeout=5)
            return
        except KafkaError:
            # Wait and retry
            time.sleep(5)
    raise TimeoutError(f"Unable to connect to Kafka after {timeout} seconds")


def delivery_report(err, msg):
    """ Called once for each message produced to indicate delivery result.
        Triggered by poll() or flush(). """
    if err is not None:
        print('Message delivery failed: {}'.format(err), file=sys.stderr)
    else:
        print('Message delivered to {} [{}]'.format(msg.topic(), msg.partition()))


def main():
    # Wait for Kafka
    brokers = "kafka:9092"
    try:
        wait_for_kafka(brokers)
    except TimeoutError as e:
        print(e)
        return

    topic = 'random_topic'

    producer = Producer(conf)

    while True:
        message = {
            'name': 'random_name',
            'date': datetime.now().strftime('%Y-%m-%d %H:%M:%S')
        }
        producer.produce(topic, value=json.dumps(message), callback=delivery_report)
        producer.flush()
        time.sleep(10)


if __name__ == "__main__":
    main()
