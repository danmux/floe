start.sh
#!/usr/bin/env bash

# example startup commands for the demo.floe.it server.
# for your own use change the certs or remove them, change the public ip. 
./floe  \
       -tags=linux,go,couch \
       -admin=123456 \
       -host_name=h1 \
       -pub_bind=172.31.22.66:443 \
       -conf=config.yml \
       -root=/home/ubuntu/floews \
       -pub_cert=/etc/letsencrypt/live/demo.floe.it/fullchain.pem \
       -pub_key=/etc/letsencrypt/live/demo.floe.it/privkey.pem \
       -prv_bind=127.0.0.1:8080