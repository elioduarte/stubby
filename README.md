# Stubby

This is a customizable proxy server that developers can use locally. 
It enables them to record network requests and use those recordings as test stubs.

## Options

```bash
stubby -help
Usage of stubby:
  -base-url string
        base URL for the application (default "http://localhost:4444")
  -config-file value
        path to the config file
  -http-port int
        port to listen on for HTTP requests (default 4444)
  -ignore-paths value
        list of paths prefixes that should not be proxied (eg: /otlp/traces)
  -stub-dir string
        directory to save the stub files (default "stubs")
  -verbose
        verbose
  -version
        display version and exit
```

## Proxy

The proxy has 3 modes: *forward*, *record* and *replay*.
The forward mode is enabled by default.

### Forward Mode

The forward mode is enabled sending a *POST* request to the proxy `/_/forward` endpoint.

The web application depends on various backend services located on different hosts. 
The proxy must determine which backend service to forward the request to based solely on the request itself. 
This is achieved by configuring the proxy to forward requests to different hosts depending on the prefix of the request path. 
Stubby comes with a preconfigured setup, but it can also be started with a different configuration file. 
The configuration file for the default settings can be found in the `./cmd/api/config.json` file.

Configuration example:

```json

{
  "default": {
    "url": "https://gw-staging.hellofresh.com"
  },
  "prefixes": [
    {
      "url": "https://translations-service.staging-k8s.hellofresh.io",
      "prefix": "/translations-service"
    }
  ]
}
```

That means that a request to `http://localhost:444/auth/login` will be forward to `https://gw-staging.hellofresh.com/auth/login`
and the request `http://localhost:4444/translations-service/translations` to `https://translations-service.staging-k8s.hellofresh.io/translation-service/translations`.

### Record Mode

The record mode is enabled sending a *POST* request to the proxy `/_/record/<profile-name>` endpoint with the *profile* name.

When enabled, every intercepted request will be saved to disk in the `stubs/<profile>/<filename>.json` file. 
The profile will be converted to lowercase, and the request path will be lowercase and have any '/' replaced with '--'. 
For instance, if the profile is "conversions" and the request path is "/gw/customer-attributes-service/attributes", the file ".stubs/conversions/gw--customer-attributes-service--attributes.json" will be created. 
If there are multiple records with the same profile and request path, they will be saved in the same file.

Stub file example:

```json
{
  "stubs": [
    {
      "request": {
        "method": "GET",
        "host": "gw-staging.hellofresh.com",
        "pathname": "/gw/customer-attributes-service/attributes",
        "query": {
          "locale": "en-US",
          "country": "us"
        }
      },
      "response": {
        "statusCode": 200,
        "body": {}
      }
    },
    {
      "request": {
        "method": "GET",
        "host": "gw-staging.hellofresh.com",
        "pathname": "/gw/customer-attributes-service/attributes"
      },
      "response": {
        "statusCode": 200,
        "body": {}
      }
    }
  ]
}
```

### Replay Mode

The replay mode is enabled sending a *POST* request to the proxy `/_/replay/<profile-name>` endpoint with the *profile* name.

When a profile is activated, the proxy will scan all the stub files in the given profile directory and create a hash table used later in the matches.
In Typescript, it would be like this: `Map<RequestHash,StubRecord[]>`.

A key will be generated for each record in the format `#<host>#<method>#<pathname>#<query>#`.

- The key will be converted to lowercase.
- If the query is defined, it will be sorted alphabetically, the values will be escaped and then joined together using = between the key and value, and & between different keys.
- If the query is not defined, value will be `*` will be used instead.

During request interception, the proxy follows the following matching procedure:

1. Calculate the complete request key, which is `#<host>#<method>#<pathname>#<query>#`. Search for this key in the hash table and return the response if found.
2. Calculate the request key without a query, which is `#<host>#<method>#<pathname>##`. Search for this key in the hash table and return the response if found.
3. Calculate a request key with any query, which is `#<host>#<method>#<pathname>#*#`. Search for this key in the hash table and return the response if found.
4. Forward the request to the staging environment and return the response.
5. Each time a key is found, it is counted. This count is used to generate different responses for the same request. The stubs are read in sequential order, so the records at the end of the array represent later requests. 

For example, given the stub file:

```json
{
  "stubs": [
    {
      "request": {
        "method": "GET",
        "host": "gw-staging.hellofresh.com",
        "pathname": "/gw/customer-attributes-service/attributes",
        "query": {
          "locale": "en-US",
          "country": "us"
        }
      },
      "response": {
        "statusCode": 200,
        "body": {}
      }
    },
    {
      "request": {
        "method": "GET",
        "host": "gw-staging.hellofresh.com",
        "pathname": "/gw/customer-attributes-service/attributes",
        "query": {}
      },
      "response": {
        "statusCode": 201,
        "body": {}
      }
    },
    {
      "request": {
        "method": "GET",
        "host": "gw-staging.hellofresh.com",
        "pathname": "/gw/customer-attributes-service/attributes"
      },
      "response": {
        "statusCode": 202,
        "body": {}
      }
    },
    {
      "request": {
        "method": "GET",
        "host": "gw-staging.hellofresh.com",
        "pathname": "/gw/customer-attributes-service/attributes"
      },
      "response": {
        "statusCode": 203,
        "body": {}
      }
    }
  ]
}
```

```bash
# The first stub will match if all of the specified query keys are passed
curl -X GET http://localhost:4444/gw/customer-attributes-service/attributes?locale=en-us&country=us
HTTP/1.1 200 OK
...

# The second stub will match if the query is not provided
curl -X GET http://localhost:4444/gw/customer-attributes-service/attributes
HTTP/1.1 201 OK
...

# The third stub will only match if the query is not the specified ones and it is the first request in that manner
curl -X GET http://localhost:4444/gw/customer-attributes-service/attributes?missing-header=true
HTTP/1.1 202 OK
...

# The fourth stub will match if the query is not the specified ones
curl -X GET http://localhost:4444/gw/customer-attributes-service/attributes?another-missing=true
HTTP/1.1 203 OK
...
```

### Skip Paths

The proxy can be set up to not forward certain endpoints, like the `/gw/otlp` endpoint. 
In these instances, the proxy will respond with a success message without forwarding the requests.

### Endpoints

A successful request to all endpoints returns the same response format:

```json
{
  "profile": "profile-name",
  "status": "Forwarding",
  "targets": {
    "default": {
      "url": {
        "scheme": "https",
        "Host": "gw-staging.hellofresh.com"
      }
    },
    "prefixes": [
      {
        "url": {
          "scheme": "https",
          "Host": "remote-config-service.staging-k8s.hellofresh.io",
        },
        "prefix": "/remote-config-service"
      }
    ]
  }
}
```

#### Status

```bash
GET /_/status
```

#### Recording Profile

```bash
POST /_/record/:profile
```

#### Replaying Profile

```bash
POST /_/replay/:profile
```

#### Forwarding

```bash
POST /_/forward
```