# Cluster Autoscaler on CuCloud
The cluster autoscaler on CuCloud scales worker nodes within any specified autoscaling group. It will run as a `Deployment` in your cluster. This README will go over some of the necessary steps required to get the cluster autoscaler up and running.

## Kubernetes Version
Cluster autoscaler must run on v1.18.6 or greater.

## Deployment Specification

### 1 ASG Setup (min: 1, max: 50, ASG Name: cu-nodepool)
```
kubectl apply -f examples/cluster-autoscaler-deployment.yaml
```

## Common Notes and Gotchas:
- By default, cluster autoscaler will not terminate nodes running pods in the kube-system namespace. You can override this default behaviour by passing in the `--skip-nodes-with-system-pods=false` flag.
- By default, cluster autoscaler will wait 10 minutes between scale down operations, you can adjust this using the `--scale-down-delay` flag. E.g. `--scale-down-delay=5m` to decrease the scale down delay to 5 minutes.
- If you're running multiple ASGs, the `--expander` flag supports three options: `random`, `most-pods` and `least-waste`. `random` will expand a random ASG on scale up. `most-pods` will scale up the ASG that will schedule the most amount of pods. `least-waste` will expand the ASG that will waste the least amount of CPU/MEM resources. In the event of a tie, cluster-autoscaler will fall back to `random`.
- If you're managing your own kubelets, they need to be started with the `--provider-id` flag.
