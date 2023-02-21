# GPU Sharing performance comparison

In this demo we compare different GPU-sharing technologies in terms of how they affect the performance
of workloads running on the same shared GPU. We benchmark the following technologies:

* Multi-Process Service (MPS)
* Multi-Instance GPU (MIG)
* Time-slicing

We measure the performance of each GPU-sharing techniques by running a set of Pods on the same shared GPU.
Each Pod has a simple container that constantly runs
inferences on a [YOLOS](https://huggingface.co/hustvl/yolos-small) model with a sample input image. The execution time
of each inference is collected and exported by Prometheus, so that later it can be easily queried and visualized.

We execute this experiment multiple times, each time with a different number of Pods running on the same GPU
(1, 3, 5 and 7), and we repeat this processes for each GPU-sharing technology.

![results](results-chart.png)

## Experimental setup

### Environment
We run all the experiments on an AKS cluster running Kubernetes v1.24.9 on 2 nodes:

* 1x [Standard_B2ms](https://learn.microsoft.com/en-us/azure/virtual-machines/sizes-b-series-burstable)
  (2 vCPUs, 8 GB RAM) - System node pool required by AKS.
* 1x [Standard_NC24ads_A100_v4](https://learn.microsoft.com/en-us/azure/virtual-machines/nc-a100-v4-series)
  (24 vCPUs, 220 GB RAM,
  1x [NVIDIA A100 80GB PCIe](https://www.nvidia.com/content/dam/en-zz/Solutions/Data-Center/a100/pdf/PB-10577-001_v02.pdf))

On the GPU-enabled node, we installed the following components:

* NVIDIA Container Toolkit: 1.11.0
* NVIDIA Drivers: 525.60.13

### Benchmarks client
We run the benchmarks by creating a Deployment with a single Pod running the [Benchmarks Client](client)
container. We created a different deployment for each GPU sharing technology. In each deployment, 
the benchmarks container always request a GPU slice with 10 GB of memory. The name of the resource requested by 
the benchmarks container depends on the specific GPU sharing technology:

* MIG: `nvidia.com/mig-1g.10gb: 1`
* MPS:  `nvidia.com/gpu-10gb: 1`
* Time-slicing: `nvidia.com/gpu.shared: 1`

We controlled how many containers were running on the same GPU by adjusting the number of replicas of 
the [Deployment](manifests/base/deployment-client.yaml).

The benchmarks client consists of a simple script that saturates constantly running 
inferences on a [YOLOS-small](https://huggingface.co/hustvl/yolos-small) model.

### Results collection
For each GPU sharing technology and for each number of Pods sharing the same GPU,
we collected the results following these steps:

1. Enable the specific GPU sharing technology on the Node
2. Create the benchmark client Pods requesting GPU slices
3. Wait for around 3 minutes
4. Collect the average inference time over the last 2 minutes


## Results

The table shows the average inference time (in seconds) for each GPU sharing technologies according to the number of
Pods sharing the same GPU.

|              | 7 Pods              | 5 Pods             | 3 Pods             | 1 Pod               |
|--------------|---------------------|--------------------|--------------------|---------------------|
| Time-slicing | 0.6848803898344092  | 0.4889516484510863 | 0.2931403323839484 | 0.08815048247208879 |
| MPS          | 0.3198182253208076  | 0.2408641491144998 | 0.1640018804617158 | 0.08796154366177533 |
| MIG          | 0.34424962169772633 | 0.3453250103939391 | 0.3413140464128765 | 0.34236589893939284 |

## How to run experiments

### Prerequisites

```bash
make install
```

This installs the following components:

- [Cert Manager](https://cert-manager.io/)
- [Kube Prometheus](https://github.com/prometheus-operator/kube-prometheus)
- [Nebuly NVIDIA Device Plugin](https://github.com/nebuly-ai/k8s-device-plugin)
- [Nos](https://github.com/nebuly-ai/nos)
- [NVIDIA GPU Operator](https://github.com/NVIDIA/gpu-operator)

### Deploy Time-slicing benchmarks client

To run multiple instances of the benchmark client on the same GPU shared using time-slicing, run 
the following command:
```bash
make deploy-ts
```

### Deploy MPS benchmarks client
To run multiple instances of the benchmark client on the same GPU shared using MPS, follow the steps below.

1. Enable dynamic MPS partitioning for the node
    ```bash
    kubectl label nodes <gpu-node> nos.nebuly.com/gpu-partitioning=mps
    ```
2. Create a Deployment with the benchmark Pod requesting MPS resources
    ```bash
    make deploy-mps
    ```
3. Wait a few seconds until `nos` GPU partitioner kicks in and automatically creates the 
requested MPS resources and Pods get scheduled.

### Deploy MIG benchmarks client
To run multiple instances of the benchmark client on the same GPU shared using MIG, follow the steps below.

1. SSH to the GPU node and enable MIG-mode
   ```bash 
    sudo nvidia-smi -i 0 -mig 1
   ```
2. Enable dynamic MIG partitioning for the node
    ```bash
    kubectl label nodes <gpu-node> nos.nebuly.com/gpu-partitioning=mig
    ```
3. Create a Deployment with the benchmark Pod requesting MIG resources
    ```bash
    make deploy-mig
    ```
4. Wait a few seconds until `nos` GPU partitioner kicks in and automatically creates the
   requested MIG devices and Pods get scheduled.

### Fetch results
You can use the Prometheus UI to visualize and explore collected results. In order to access it, run the following 
command to port-forward you local port to the Prometheus instance running in the cluster:
```bash
make port-forward-prometheus
```

You can then access Prometheus UI at [localhost:9090](http://localhost:9090).

From the Prometheus UI you can get the average inference time over the last 2 minutes by running the following query:
```bash
avg(sum(rate(inference_time_seconds_sum[2m])) / sum(rate(inference_time_seconds_count[2m])))
```

