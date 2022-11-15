## Configuration
TODO

## TODO

### How it works
Nebulnetes deploys the **MIG Agent** component on every node labelled with `n8s.nebuly.ai/auto-mig-enabled: "true"`.
The MIG Agent exposes to Kubernetes the MIG geometry of all the GPUs of the node on which it is deployed by using
the following Node annotations:
* `n8s.nebuly.ai/status-gpu-<index>-<mig-profile>-free: <quantity>`
* `n8s.nebuly.ai/status-gpu-<index>-<mig-profile>-used: <quantity>`

These annotations are used by the **MIG GPU Partitioner** component for reconstructing the current state of the cluster
and deciding how to change the MIG geometry of its GPUs. It does that everytime there is a batch of pending Pods that
cannot be scheduled due to lack of MIG resources: it processes the batch and tries to update the MIG
geometry the available GPUs by deleting unused profiles and creating the ones required by the pending pods. If the
updated geometry allows to schedule one or more pods, the GPU Partitioner applies it by updating the spec annotations
of the involved nodes. These annotations have the following format:
`n8s.nebuly.ai/spec-gpu-<index>-<mig-profile>: <quantity>`

The MIG Agent watches node's annotations and everytime there desired MIG partitioning (specified with the spc annotations
mentioned above) does not match the current state, it tries to apply it by creating and deleting the MIG profiles
on the target GPUs.

Note that MIG profiles being used by some pod are never deleted: before applying the desired status the MIG agent
always checks whether the profiles to delete are currently allocated.

