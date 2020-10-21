while :
do
    sudo -E ./distributed-service-discovery -d -p 22 -o mysql
    sleep 30
done