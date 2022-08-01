# Purpose of the demo
The goal of this demo is to run the [cluster-autoscaler](https://github.com/kubernetes/autoscaler/tree/master/cluster-autoscaler) with the Azure provisioner on a local k8s cluster run using [kind](https://kind.sigs.k8s.io/), in order to see whether the autoscaler can provision new nodes on an Azure VMSS even when running in a k8s cluster outside of Azure cloud.

# How to run
1. Create a kind cluster from the respective config file
```
kind create cluster --config kind.yaml --name anton
```

2. Create an Azure Service principal with role "Owner" over your Azure Subscription
```
az ad sp create-for-rbac --role="Contributor" --scopes="/subscriptions/<your-subscription-id>" --output json
```
Copy the output of the command, you will need it later.

3. Move to the [terraform](terraform) directory and create a file backend.tfvars containing the credentials for connecting to Azure. You can use the file backend.tfvars.example as template
```
subscription_id=""
tenant_id=""
client_id=""
client_secret=""
```

4. Still inside the terraform directory, run terraform apply for provisioning the required cloud infrastructure:
```
terraform apply --var-file="backend.tfvars"
```

5. Get the client ID and the client secret of the cluster-autoscaler identity created by Terraform:
```
terraform output cluster_autoscaler_sp
```

6. Move to this directory and create a file cluster-autoscaler-secrets.yaml, starting from the template cluster-autoscaler-secrets.yaml.example. Fill the missing values using the base64 encoding of the credentials fetched in the previous step
```
echo "<value>" | base64
```

7. Deploy the cluster-autoscaler:
```
kubectl create -f cluster-autoscaler.yaml
kubectl apply -f cluster-autoscaler-secrets.yaml
```

# Expected results
Creating a deployment that exceeds the kind cluster resources (see [dummy-deployment.yaml](dummy-deployment.yaml))
should make the cluster-autoscaler increase the number of VM instances of the Azure VMSS. 

Since networking is not configured properly, the new provisioned nodes on the VMSS should not be able to join 
the kind cluster.

# Actual results
The cluster-autoscaler does not provision any new VM instance in the VMSS.

### Problems
Azure provider of cluster-autoscaler uses the nodes of the cluster in order to replicate them to the VMSS, and it expects them to have a provider ID in the following format: azure:/<location>/<name>.

Since if the cluster is not running on Azure the provider ID is different, the cluster autoscaler cannot fetch info about the nodes and the scaling fails with the following error msg:
```
E0614 16:55:03.345723       1 static_autoscaler.go:290] Failed to get node infos for groups: "kind://docker/anton/anton-control-plane" isn't in Azure resource ID format
```

This could be due to a bug in the cluster-autoscaler integration with Azure, please refer to the [issue](https://github.com/kubernetes/autoscaler/issues/4972) opened on GitHub.


