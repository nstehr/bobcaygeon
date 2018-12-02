# Bobcaygeon Frontend

Provides an [Envoy](https://www.envoyproxy.io/) that frontends the API management servers.  It also contains a small application to act as an xDS control plane for Envoy to handle the service discovery and serve a web application that consumes the management API

# Basic Design
## Proxy
The proxy provides two basic functions:
1. Frontends the Bobycaygeon API management servers.  There can be one or more management servers and the proxy will round robin between all of them.  This frontend app will know about the API management servers by being a participant of the memberlist cluster and providing the management server addresses to the proxy dynamically by acting as a control plane for Envoy xDS APIs
2. Translation for the web app to be able to use [grpc-web](https://github.com/grpc/grpc-web)

## Application
The application itself will be responsible for:
1. Serving the static content that makes up the web app
2. Providing dynamic routing configuration to the envoy instance