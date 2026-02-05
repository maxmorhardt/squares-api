while true; do
    kubectl port-forward svc/nats 4222:4222 -n nats
done