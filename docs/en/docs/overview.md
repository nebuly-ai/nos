# Overview

nos is the open-source module for running AI workloads on Kubernetes in an optimized way, 
both in terms of hardware utilization and workload performance.

The operating system layer is responsible for workloads scheduling and hardware abstraction. 
It orchestrates the workloads taking into account considerations specific for AI/ML workloads and leveraging techniques 
typical of High-performance Computing (HPC), and it hides the underlying hardware complexities.

Currently, this layer provides two features [Dynamic GPU partitioning](dynamic-gpu-partitioning.md) and
[Elastic Resource Quota management](elastic-quota.md).
