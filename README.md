# go-http2mqtt

go-http2mqtt is go daeamon (or go library http2mqtt) useful to talk to mqtt protocol throught REST API. it supports

  - publish data to topic
  - subscribe to topic receiving data via sse stream

You can also:
  - configure partially the tool from API

It can be used to as a quick integration of mqtt language in a web application. 

### Installation of the daemon
Install the dependencies and start the server.

```sh
$ go get
$ cd  cmd/go-http2mqtt/
$ go build
```
to start deamon (note port is required in the host)

```sh
$ cd  cmd/go-http2mqtt/
$ ./go-http2mqtt <host:port> <mqttbroker:port>
```
if you want to authenticate the API with user:password
```sh
$ cd  cmd/go-http2mqtt/
$ ./go-http2mqtt <host:port> <mqttbroker:port> -u $user -p $password
```


### Endpoints

go-http2mqtt exposes the following endpoints:
(in case of port 8000 on localhost)
```
http://localhost:8000/ping
http://localhost:8000/publish
http://localhost:8000/subscribe
http://localhost:8000/broker
http://localhost:8000/streams
```


##### localhost:8000/ping {GET}
This is the only always API NOT authenticated: it is a simple check to query a reply "pong"
```sh
$ curl -i -X GET -H "Content-Type: application/json" http://localhost:8000/ping
```
##### localhost:8000/publish {POST}
the json payload is:
```json
{
    "topic": "/topic/1",
    "data": "this is a test",
    "qos" : 0,
    "retained" : "true"
}
```
"qos and "retained" fields are optionals

```sh
curl -u $user:$pass -i -X POST -H "Content-Type: application/json" -d '{"topic":""/topic/1" , "data": "this is a test"}' http://localhost:8000/publish
```
##### localhost:8000/subscribe {POST}
the json payload is:
```json
{
    "topic": "/topic/1",
    "qos" : 0,
}
```
"qos is optional
```
curl -u $user:$pass -i -X POST -H "Content-Type: application/json" -d '{"topic": "/topic/1" , "qos": 0}' http://localhost:8000/subscribe
```
##### localhost:8000/broker{GET}
To get info about subscriptions and other broker's stuff:

```
curl -u $user:$pass curl -i -X GET -H "Content-Type: application/json" http://localhost:8000/broker
```
```json
{
    "broker":"localhost:1883",
    "connected":true,
    "subscriptions":[{"topic":"/topic/1","qos":0}],
    "user":null
}
```

##### localhost:8000/streams{GET}
sse stream to receive messages of the subscribed topics 
event:/topic/1
data:"eyJjb21tYW5kIjoicmVzZXQifQ=="

```
curl -u $user:$pass curl -i -X GET -H "Content-Type: application/json" http://localhost:8000/streamsr
```

### Development

Want to contribute? Great! 

### Todos

 - Write Tests
 - Add Swagger Doc

License
----

MIT


**Free Software!**

[//]: # (These are reference links used in the body of this note and get stripped out when the markdown processor does its job. There is no need to format nicely because it shouldn't be seen. Thanks SO - http://stackoverflow.com/questions/4823468/store-comments-in-markdown-syntax)


   [dill]: <https://github.com/joemccann/dillinger>
   [git-repo-url]: <https://github.com/joemccann/dillinger.git>
   [john gruber]: <http://daringfireball.net>
   [df1]: <http://daringfireball.net/projects/markdown/>
   [markdown-it]: <https://github.com/markdown-it/markdown-it>
   [Ace Editor]: <http://ace.ajax.org>
   [node.js]: <http://nodejs.org>
   [Twitter Bootstrap]: <http://twitter.github.com/bootstrap/>
   [jQuery]: <http://jquery.com>
   [@tjholowaychuk]: <http://twitter.com/tjholowaychuk>
   [express]: <http://expressjs.com>
   [AngularJS]: <http://angularjs.org>
   [Gulp]: <http://gulpjs.com>

   [PlDb]: <https://github.com/joemccann/dillinger/tree/master/plugins/dropbox/README.md>
   [PlGh]: <https://github.com/joemccann/dillinger/tree/master/plugins/github/README.md>
   [PlGd]: <https://github.com/joemccann/dillinger/tree/master/plugins/googledrive/README.md>
   [PlOd]: <https://github.com/joemccann/dillinger/tree/master/plugins/onedrive/README.md>
   [PlMe]: <https://github.com/joemccann/dillinger/tree/master/plugins/medium/README.md>
   [PlGa]: <https://github.com/RahulHP/dillinger/blob/master/plugins/googleanalytics/README.md>
