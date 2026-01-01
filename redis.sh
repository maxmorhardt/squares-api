while true; do
    echo "Starting Redis port-forward..."
    kubectl port-forward svc/redis-master 6379:6379 -n db
    echo "Port-forward disconnected, restarting in 2 seconds..."
    sleep 2
done