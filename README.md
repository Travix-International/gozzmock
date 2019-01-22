# gozzmock
Mock server in go!

Inspired by http://www.mock-server.com/

[![Build Status](https://travis-ci.org/Travix-International/gozzmock.svg?branch=master)](https://travis-ci.org/Travix-International/gozzmock)
[![Coverage Status](https://coveralls.io/repos/github/Travix-International/gozzmock/badge.svg?branch=master)](https://coveralls.io/github/Travix-International/gozzmock?branch=master)
[![License](https://img.shields.io/github/license/Travix-International/gozzmock.svg)](https://github.com/Travix-International/gozzmock/blob/master/LICENSE)
[![Go Report](https://goreportcard.com/badge/github.com/Travix-International/gozzmock)](https://goreportcard.com/report/github.com/Travix-International/gozzmock)

# Docker Hub
https://hub.docker.com/r/travix/gozzmock/

# Description
Travix uses gozzmock to avoid dependencies on 3rd party services at test environment.
Gozzmock is a "transparent" mock and fully manageable trow API calls. Transparency means, some calls can be mocked, other calls will be send to real endpoint. 

Package manager: [dep](https://github.com/golang/dep)


# Install
```
 docker pull travix/gozzmock
```

# Environment variables
* *GOZ_LOGLEVEL* - log level. Values: debug, info, warn, error, fatal, panic. Default: debug
* *GOZ_EXPECTATIONS* - array of expectations is json format. Default: empty. It is used to load default/forward expectations when appication starts.
* *GOZ_EXPECTATIONS_FILE* - a path to json file with expectations.
* *GOZ_PORT* - port number at local machine for http server 
* *HTTP_PROXY*, *HTTPS_PROXY* - http proxy configuration. May be either a complete URL or a "host[:port]". Detailed description: https://golang.org/src/net/http/transport.go

## Example
```
docker run -it -p8080:8080 --env GOZ_EXPECTATIONS="[{\"key\": \"k1\"},{\"key\": \"k2\"}]" --env GOZ_LOGLEVEL="info" travix/gozzmock
```

# Use-cases
For instance, there is a task to mock GitHub API call https://api.github.com/user
By default, /user returns
```json
{
  "message": "Requires authentication",
  "documentation_url": "https://developer.github.com/v3"
}
```
Run container with gozzmock
```
docker run -it -p8080:8080 travix/gozzmock
```

Upload a "forward" expectation to gozzmock. Expectation structure:
```json
{
    "key": "forwardExpectation",
    "forward": {
        "host": "api.github.com",
        "scheme": "https"
    }
}
```
Expectation should be sent to /gozzmock/add_expectation endpoint like this
```bash
curl -d '{"forward":{"host":"api.github.com","scheme":"https"},"key":"forwardExpectation"}' -X POST http://192.168.99.100:8080/gozzmock/add_expectation
```
*NOTE* 192.168.99.100 - ip of host machine

To validate that expectation works
```bash
curl http://192.168.99.100:8080/user
```
returns
```json
{
  "message": "Requires authentication",
  "documentation_url": "https://developer.github.com/v3"
}
```

Add expectation with response:
```json
{
    "key": "responseExpectation",
    "request": {
        "method": "GET",
        "path": "mocked"
    },
    "response": {
        "body": "response from gozzmock",
        "headers": [
            {
                "Content-Type": "text/plain; charset=utf-8"
            }
        ],
        "httpcode": 200
    },
    "priority": 1
}
```

```bash
curl -d '{"key":"responseExpectation","request":{"method":"GET","path":"mocked"},"response":{"body":"response from gozzmock","headers":[{"Content-Type":"text/plain; charset=utf-8"}],"httpcode":200},"priority":1}' -X POST http://192.168.99.100:8080/gozzmock/add_expectation
```

Send request with "mocked" in path:
```bash
curl http://192.168.99.100:8080/user?arg=mocked
curl http://192.168.99.100:8080/user?mocked
curl http://192.168.99.100:8080/user/mocked
```

For all those request response will be from expectation:
```
response from gozzmock
```

To remove expectation, send request to /gozzmock/remove_expectation with structure:
```json
{
    "key": "responseExpectation"
}
```

```bash
curl -d '{"key":"responseExpectation"}' -X POST http://192.168.99.100:8080/gozzmock/remove_expectation
```

# Expectations structure

## Root level 
* key - unique identifier for message. If another expectation is added with same key, original will be replaced
* priority (optional) - is used to define order. First expectation has greatest priority.
* dealy (optional) - delay in seconds before sending response
* request - block of filters/conditions for incoming request
* response - this block will be sent as response if incoming request passes filter in "request" block
* forward - this block describes forwarding/proxy. If incoming request passes filter in "request" block, request will be re-sent according to "forward" block.

*NOTE* only one block should be set: response or forward

## Request
Structure of "request" block
* method - HTTP method: POST, GET, ...
* path - path, including query (?) and fragments (#) 
* body - request body
* headers - headers in request

*NOTE* It is allowed to use regex as well as simple string.
For instance, if path: ".*" - it will be parsed as regex. if string "abc" - it will be used as substring

## Forward
Structure of "forward" block
* Scheme - HTTP or HTTPS
* host - target host name. Host name of original request will be replaced with this value. Path and query will be same.
* headers - headers which will be added/replaced when forwarding

## Response
Structure of "response" block
* method - HTTP method: POST, GET, ...
* path - path, including query (?) and fragments (#) 
* body - response body
* headers - headers in response


## Endpoints
* /gozzmock/status - status and readiness endpoint
* /gozzmock/add_expectation - add or update an expectation
* /gozzmock/remove_expectation - remove expectation by key
* /gozzmock/get_expectations - get list of all stored expectations


#TODO
Add example with js