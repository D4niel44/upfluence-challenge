# Aggregation Service

This repository implements a simple aggregation service built on top of  https://stream.upfluence.co/stream api. The product requirements for this service can be seen here  https://gist.github.com/AlexisMontagne/fc105d35fdfb7c3c32213e78b9cadad3.

## Design
The service was implemented in Golang, leveraging its modern concurrency features, like channels.

### API Definition

The service supports GET requests on the endpoint `/analysis`. Requests on any other endpoint, or with a different method will result in a 404 error.

The endpoint requires two query parameters.

- `duration`: A timespan during which the service will aggregate any incoming post (i.e. 5s, 10m, 2h).

- `dimension`: A field over which the average value will be calculated (i.e. likes, comments, favorites). Any numeric field is supported.

For a valid request, the endpoint will generate a response with the following data.

- **Total number of posts analyzed:** A post is considered any event that has a root key (events of the form `data: {}`) are ignored for this an all other metrics. This metrics also counts posts without the requested dimension.

- **Minimum timestamp:** The minimum timestamp of all the posts analyzed. This includes post without the requested metric.

- **Maximum timestamp:** The maximum timestamp of all the posts analyzed. This includes post without the requested metric.

- **Average of the dimension:** Posts that do not contain the given dimension are ignored when calculating this metric. 

**Note:** An alternative definition could be to only count posts with the given dimension for all metrics, which could be better depending on the use case.

An example response is shown below.

```
< HTTP/1.1 200 OK
< Cache-Control: no-cache
< Content-Type: application/json
< Date: Tue, 13 Dec 2022 01:05:31 GMT
< Content-Length: 113
<
{
    "avg_likes" : 3091.020310633202
    "maximum_timestamp" : 1670893075,
    "minimum_timestamp" : 1153264746,
    "total_posts": 15102
}
```

### Architectural Overview

A straightforward way to implement a service that aggregates events from the upfluence stream api would be to create a new request to the SSE api each time a request is made, aggregate the events that are sent, and return the results to the caller. However, the events sent by the upfluence streams for multiple concurrent requests will be the same, so there is no need to fetch it more than once.

The idea behind the design is to have a unique connection to the events api, and then forward the events to each request handler, which aggregates the values and returns a response with the results. The proposed design consists of three layers, which can be seen in the following diagram.

![Architecture](/img/Architecture.png "Architecture")

As can be seen above the service is split into three components.

- SSE Client: This component keeps a unique connection with the SSE api, and forwards the events to the Broadcast Service by using a channel.

- Broadcast Service: This component forwards the events it receives te each registered client. It also supports performing an operation on each event before forwarding them. Clients can register themselves to start receiving events and unregister when they are no longer interested in receiving them. 

- Aggregation query handler: This component handles the user requests, aggregates the events received by the Broadcast service, and returns the results to the user.

### Ways to Improve

There are many ways the service could be improved. A few are mentioned below.

- Golang provides a mechanism to recover from panics and avoid crashing the whole service. This was not leveraged due to time constraints. However, the current implementation allows recovering in the most likely error scenario, which is when the connection to the SSE remote api is lost.

- Currently, if the service losses connection with the SSE api, it will try to reconnect, but it will not notify request handlers that the connections to the events source has been lost. If a request occurs during such scenario, the end user will not be able to differentiate the results, which will be a default aggregation query, from a result were the was healthy, but there were no events.

- Currently, there are several hardcoded configuration values, like the port, or the url of the upfluence sse api. A way to improve the implementation would be to allow to set this values at runtime, instead of compile time, by either passing them as command line arguments, or using a configuration file.

- The current implementation of the broadcast service relies on using a shared map, and locking every time a clients wants to start/stop receiving events. This will cause a blotleneck if there are many aggregation requests made at the same time. One way to improve this would be to use the `sync.Map` type,
which is optimized for concurrent use, instead of the builtin map. Another approach could be to remove all shared state, and instead use channels for adding/removing clients from receiving events.

## Compile and Run

Go version 1.19 is required to compile and run the service. No external dependencies are used.

A Makefile is provided to help with compilation. The following commands can be used to build and test.

- To build the project

```
make build
```

- To run the tests in the project

```
make test
```

When the project finishes building, the executable that starts the service will be located at `bin/aggregation-service`, which does not require any arguments to work.