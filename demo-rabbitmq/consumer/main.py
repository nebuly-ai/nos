import os
import logging
import random
import sys
import time

import pika


def callback(ch, method, properties, body):
    logging.info(f'[x] Received "{body}"')
    sleep_seconds = random.randint(1, 10)
    logging.info(f"Sleeping for {sleep_seconds} seconds...")
    time.sleep(sleep_seconds)
    ch.basic_ack(delivery_tag=method.delivery_tag)
    logging.info("Acknowledging ")


def main():
    host = os.environ["RABBITMQ_HOST"]
    port = os.environ["RABBITMQ_PORT"]
    queue = os.environ["RABBITMQ_QUEUE"]

    connection = pika.BlockingConnection(pika.ConnectionParameters(host, port=int(port)))
    channel = connection.channel()

    channel.basic_consume(queue=queue, on_message_callback=callback, auto_ack=False)

    try:
        logging.info("[*] Waiting for messages. To exit press CTRL+C")
        channel.start_consuming()
    except KeyboardInterrupt:
        logging.info("Interrupted")
        sys.exit(0)


if __name__ == '__main__':
    logging.basicConfig(
        level=logging.INFO,
        format='[%(asctime)s] {%(pathname)s:%(lineno)d} %(levelname)s - %(message)s',
        datefmt='%H:%M:%S'
    )
    main()
