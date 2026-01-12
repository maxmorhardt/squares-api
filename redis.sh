while true; do
    kubectl port-forward svc/redis-master 6379:6379 -n redis
done