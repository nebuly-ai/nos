# Demo RabbitMQ

The following is a demo of using [RabbitMQ](https://www.rabbitmq.com/) as message broker for submitting
messages on a queue and splitting out the work among different consumers, each of which processes a different part
of the queue, like illustrated in the image below.

![](https://www.rabbitmq.com/img/tutorials/python-two.png)

The demo is organized as follows:

* [consumer](consumer): contains a Python script that consumes the messages from a queue
* [producer](producer): contains a Go application that writes messages on a Queue every n seconds
* [manifests](manifests): directory containing the k8s manifests required for deploying the demo (e.g. the consumer and
  the
  producer applications)

### Message persistence

The messages written by the [Producer](producer) on the Queue are persisted by the RabbitMQ cluster, so that if one or
more
node fails the messages are not lost. This is done by configuring the Queue as "durable" and by using the "persistent
delivery mode"
for sending messages.

### Message acknowledge

The consumers have to acknowledge each message they process successfully. In this way, if a consumer receive a message
and then
crashes before fully processing it, the message won't be deleted and instead it will be re-queued again after a certain
timeout.
If this way, if a worker dies, all its pending messages will be delivered to another worker. 

This is done by configuring the queue for [manual acknowledge](https://www.rabbitmq.com/confirms.html), which means that 
each consumer has to explicitly acknowledge each message it processes successfully.

In order to distribute more evenly the messages between the consumers, the channel used by each consumer for
connecting to the queue is configured with ``prefetch_count=1``, so that
_RabbitMQ doesn't dispatch a new message to a worker until it has processed and acknowledged the previous one_

### RabbitMQ installation

RabbitMQ is deployed on a KIND cluster using
the [RabbitMQ Cluster Operator](https://www.rabbitmq.com/kubernetes/operator/operator-overview.html).

Moreover,
the [RabbitMQ Messaging Topology Operator](https://www.rabbitmq.com/kubernetes/operator/install-topology-operator.html)
is used for specifying users and permissions using k8s manifests.

## How to run the demo

1. Create a KIND cluster

```shell
make setup
```

2. Install the Operators

```shell
make install
```

3. Build the producer and consumer Docker images and load them into the KIND cluster

```shell
make build
```

4. Deploy the RabbitMQ cluster and the producer and consumer applications

```shell
make deploy
```

5. Cleanup

```shell
make clean
```

### Access RabbitMQ admin portal

After you deploy the RabbitMQ cluster, you can access its UI from a web browser by port forwarding from your local
machine
to the RabbitMQ service:

```shell
kubectl port-forward service/my-cluster 15672
```

The UI is then available at http://localhost:15672. You can get the admin login credentials by running:

```shell
make credentials
```