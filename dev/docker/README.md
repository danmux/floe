from the docker folder - generate your key pair

```
ssh-keygen -N "" -t rsa -b 4096 -C "build@floe.it" -f deploy_rsa
```

add the public key deploy_rsa.pub to the github repo deploy key 