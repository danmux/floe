
http://127.0.0.1:8080/build/api/config
Cookie:floe-sesh=547dd44e09ac2835

###

http://127.0.0.1:8080/build/api/flows
Cookie:floe-sesh=547dd44e09ac2835

###

http://127.0.0.1:8080/build/api/runs/archive
Cookie:floe-sesh=547dd44e09ac2835

###

POST http://127.0.0.1:8080/build/api/login HTTP/1.1
content-type: application/json

{
    "user": "admin",
    "password": "password"
}