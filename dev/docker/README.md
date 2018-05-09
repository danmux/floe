Testing on Docker
=================

To test your flow in isolation from your host tooling and environment its a good plan to test in a container.

From this docker folder (`cd dev/docker`)- generate your key pair

```
ssh-keygen -N "" -t rsa -b 4096 -C "build@floe.it" -f deploy_rsa
```

Add the public key deploy_rsa.pub to the github repo deploy key (TODO link)

Rebuild the image (still from this dev folder) ...

`docker build -t golang:latest .`

and shell onto it...

`docker run -it golang /bin/bash`

Do your ting...

