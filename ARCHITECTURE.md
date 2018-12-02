# Bobcaygeon Architecture

The full deployment of a bobcaygeon cluster involves three parts.

1. bcg - this acts as the airplay server itself.  A single bcg instance will be a fully functional airplay server.  By deploying multiple instances the default behaviour will be for them to join together and all play the same stream.
2. bcg-mgmt - this acts as a management API server.  It will provide the APIs neccessary to form zones of bcg servers to allow finer grain control over what servers are working together.  Can be one or more to work together in a cluster
3. bcg-frontend - will provide an [Envoy](https://www.envoyproxy.io/) that will frontend the group of bcg-mgmt servers.  Also serves a sample web application that makes use of the management APIs

By splitting it up into 3 components it allows for a more separation of concerns, bcg can focus just on being an airplay server and not be muddied with zones, or higher level multi-room functionality.  It also let's me play around a bit more with distributed system concepts :)