
#!/bin/bash

docker run --rm -p 9211:9211 -p 9901:9901 --mount type=bind,source="$(pwd)"/envoy.yml,target=/etc/envoy/envoy.yaml envoyproxy/envoy:latest