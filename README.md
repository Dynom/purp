This is a small project setup to test a Docker Swarm cluster, or any cluster with networking and load-balancing features.
This small app mimics micro-services and allows you to see how the cluster behaves, without the need of a complex setup.

You can use it for tinkering and testing :-)

# Microservices and hops

A typical micro-services architecture will request other services based on responsibility, an example could be:
```
request  -> [Gatekeeper] -> [Order service]     (Process order)
            [          ] -> [Message service]   (Send confirmation e-mail)
            [          ] -> [Content service]   (Update listing)
response <- [          ]
```
In this example, the amount of hops is 1. The public facing service accepts the request and puts the action through to 
various services in order. This is the equivalent of 1 hop with a certain work-factor.

A more complicated situation, however, might have several hops
```
request  -> [Gatekeeper] -> [Recommendation service]  (Fetches a personalised listing)
            [          ]    [                      ] -> [Recommendation process] -> [ Statistics service ]
            [          ]    [                      ] <- [                      ]
            [          ]    [                      ] 
            [          ]    [                      ] -> [Content service]  (Fetch product details)
            [          ] <- [                      ]
response <- [          ]    
```
This example has up to 3 hops.


# Example
Example local setup:

```
# Define 10 services
PORTS=$(seq 8080 8090);

# Generate the discovery
ADD_HOST_LINE="";
for PORT in ${PORTS}; do
  ADD_HOST_LINE="${ADD_HOST_LINE} --add-host=localhost:${PORT}";
done

# Start the services
for PORT in ${PORTS}; do
  ./purp ${ADD_HOST_LINE} --listen-on ${PORT} &
done

# When done. Kill everything with:
kill $(pgrep -f purp)
```

Just pick a service and specify the amount of hops:
```
curl http://localhost:8080/?hops=3
Done
```

The amount of hops specify how many services it should perform sub-requests to, each request waits until the sub-request is finished, emulating a chain.

To stress the *services* you can pick something as [wrk](https://github.com/wg/wrk), [Hey](https://github.com/rakyll/hey), [ab](https://httpd.apache.org/docs/2.4/programs/ab.html), [Siege](https://github.com/JoeDog/siege), etc. For this particular test I prefer *wrk* since you can specify load generation for a duration.

```
wrk -d 60s http://localhost:8080/?hops=0
Running 1m test @ http://localhost:8080/?hops=0
  2 threads and 10 connections
  ...
```
