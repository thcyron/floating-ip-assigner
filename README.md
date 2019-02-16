# floating-ip-assigner

**floating-ip-assigner** is a sidecar for Kubernetes which assigns a Hetzner Cloud Floating IP
to the node the container is currently running on.

## Usage

Just add the container to your pod spec:

```yaml
spec:
  containers:
  - name: floating-ip-assigner
    image: thcyron/floating-ip-assigner:1.0.1
    env:
    - name: HCLOUD_TOKEN
      value: "<token>"
    - name: HCLOUD_FLOATING_IP_ID
      value: "<id>"
```
