# Bobcaygeon Architecture

The full deployment of a bobcaygeon cluster involves three parts.

1. bcg - this acts as the airplay server itself.  A single bcg instance will be a fully functional airplay server.  By deploying multiple instances the default behaviour will be for them to join together and all play the same stream.  Airplay protocol compliant, you can connect to a bcg instance with just an iOS/OSX device that supports airplay.  Supports control of the sending application via dacp as well as parsing of now playing information using daap.
2. bcg-mgmt - this acts as a management API server.  It will provide the APIs neccessary to form zones of bcg servers to allow finer grain control over what servers are working together.  Can be one or more to work together in a cluster, using Raft for consensus.  Is the component of the cluster that provides persistent state.
3. bcg-frontend - will provide an [Envoy](https://www.envoyproxy.io/) that will frontend the group of bcg-mgmt servers.  The bcg-frontend binary acts as a control plane for envoy and also serves a sample react based web application that makes use of the management APIs.  The web application is packed into the binary using [statik](https://github.com/rakyll/statik).

## Clustering
All the `bcg-*` applications participate in the same cluster, using [memberlist](https://github.com/hashicorp/memberlist).  The nodes will discover each other using mdns.

There is an additional subcluster formed, if you run more than one `bcg-mgmt` instance.  `bcg-mgmt` instances will use raft to elect a leader and maintain state.

## API
All API communication is done over grpc.  This includes the web application.  It uses grpc-web to talk to the `bcg-mgmt` component.  Because of this, we use Envoy to proxy the requests.  Envoy actually serves two purposes, to handle the grpc-web calls, as well as loadbalance across multiple `bcg-mgmt` binaries, if more than one are running.