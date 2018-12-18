package control

import (
	"fmt"
	"log"
	"net"
	"time"

	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	hcm "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	xds "github.com/envoyproxy/go-control-plane/pkg/server"
	"github.com/envoyproxy/go-control-plane/pkg/util"
	"github.com/gogo/protobuf/types"
	"google.golang.org/grpc"
)

const (
	clusterName   = "mgmt-cluster"
	listenerName  = "listener_0"
	listenerPort  = 9211
	envoyNodeName = "bcg-frontend-envoy" // needs to match what is in the envoy config
)

type hash struct {
}

// ID function
func (h hash) ID(node *core.Node) string {
	if node == nil {
		return "unknown"
	}
	return node.Id
}

// ControlPlane represents an Envoy control plane we use to dynamically add endpoints
type ControlPlane struct {
	apiPort       int
	snapshotCache cache.SnapshotCache
	cacheVersion  int32
	cluster       *api.Cluster
	listener      *api.Listener
	endpoints     *api.ClusterLoadAssignment
}

// MgmtEndpoint represents a single endpoint of a bcg-mgmt server that will become proxied by us
type MgmtEndpoint struct {
	Host string
	Port uint32
}

// NewControlPlane will instantiate a control plane
func NewControlPlane(apiPort int) *ControlPlane {
	cp := &ControlPlane{apiPort: apiPort, snapshotCache: cache.NewSnapshotCache(false, hash{}, nil), cacheVersion: 1}

	// build our initial dyanmic configuration
	// TODO: could probably just use the EDS service,
	// and everything else be static...
	la := buildEndpoints([]MgmtEndpoint{})
	c := buildCluster()
	l := buildListener()

	cp.cluster = c
	cp.listener = l
	cp.endpoints = la

	cp.newSnapshot()

	return cp
}

// Start starts up a control plane to respond to Envoy XDS requests
func (cp *ControlPlane) Start() {
	server := xds.NewServer(cp.snapshotCache, nil)
	grpcServer := grpc.NewServer()
	lis, _ := net.Listen("tcp", fmt.Sprintf(":%d", cp.apiPort))

	discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
	api.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	api.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	api.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	api.RegisterListenerDiscoveryServiceServer(grpcServer, server)

	grpcServer.Serve(lis)
}

// UpdateEndpoints will tell Envoy to direct traffic to the MgmtEndpoints
func (cp *ControlPlane) UpdateEndpoints(mgmtEndpoints []MgmtEndpoint) {
	cp.endpoints = buildEndpoints(mgmtEndpoints)
	cp.newSnapshot()
}

// AddEndpoint adds a single endpoint
func (cp *ControlPlane) AddEndpoint(mgmtEndpoint MgmtEndpoint) {
	l := cp.endpoints.Endpoints[0].LbEndpoints
	l = append(l, buildEndpoint(mgmtEndpoint))
	cp.endpoints.Endpoints[0].LbEndpoints = l
	cp.newSnapshot()
}

// RemoveEndpoint remove a single endpoint
func (cp *ControlPlane) RemoveEndpoint(mgmtEndpoint MgmtEndpoint) {
	l := cp.endpoints.Endpoints[0].LbEndpoints
	foundIndex := -1
	for i, e := range l {
		addr := e.Endpoint.Address.GetSocketAddress().Address
		port := e.Endpoint.Address.GetSocketAddress().GetPortValue()
		if addr == mgmtEndpoint.Host && port == mgmtEndpoint.Port {
			foundIndex = i
			break
		}
	}
	if foundIndex >= 0 {
		// remove the list of endpoints
		l = append(l[:foundIndex], l[foundIndex+1:]...)
		cp.endpoints.Endpoints[0].LbEndpoints = l
		cp.newSnapshot()
	}
}

func (cp *ControlPlane) newSnapshot() {
	snapshot := cache.NewSnapshot(fmt.Sprint(cp.cacheVersion), []cache.Resource{cp.endpoints}, []cache.Resource{cp.cluster}, nil, []cache.Resource{cp.listener})
	_ = cp.snapshotCache.SetSnapshot(envoyNodeName, snapshot)
	cp.cacheVersion = cp.cacheVersion + 1
}

func buildCluster() *api.Cluster {
	//EDS, or endpoint discovery service is what we can use to dynamically add endpoints
	// that will be proxied.  Here we create a source, point it to our endpoint that
	/// represents our control plane
	edsSource := &core.ConfigSource{
		ConfigSourceSpecifier: &core.ConfigSource_ApiConfigSource{
			ApiConfigSource: &core.ApiConfigSource{
				ApiType: core.ApiConfigSource_GRPC,
				GrpcServices: []*core.GrpcService{{
					TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
						EnvoyGrpc: &core.GrpcService_EnvoyGrpc{ClusterName: "xds_cluster"},
					},
				}},
			},
		},
	}

	// creates a cluster.  A cluster in envoy terms represents a collection
	// of endpoints that can be called. We tell our cluster that
	// endpoints will be dynamically discovered, letting us control
	// service discovery
	clust := &api.Cluster{
		Name:                 clusterName,
		ConnectTimeout:       250 * time.Millisecond,
		Type:                 api.Cluster_EDS,
		EdsClusterConfig:     &api.Cluster_EdsClusterConfig{EdsConfig: edsSource},
		Http2ProtocolOptions: &core.Http2ProtocolOptions{},
	}

	return clust
}

func buildListener() *api.Listener {
	// Listener is the top level object for envoy config
	l := api.Listener{}
	l.Name = listenerName

	// address of our proxy that callers will access
	l.Address = core.Address{
		Address: &core.Address_SocketAddress{
			SocketAddress: &core.SocketAddress{
				Protocol: core.TCP,
				Address:  "0.0.0.0",
				PortSpecifier: &core.SocketAddress_PortValue{
					PortValue: listenerPort,
				},
			},
		},
	}

	r := route.Route{}
	r.Match = route.RouteMatch{
		PathSpecifier: &route.RouteMatch_Prefix{
			Prefix: "/",
		},
	}

	t := 0 * time.Second
	// this route action specifies to direct traffic we receive on this
	// listener to the specified cluster
	r.Action = &route.Route_Route{
		Route: &route.RouteAction{
			ClusterSpecifier: &route.RouteAction_Cluster{
				Cluster: clusterName,
			},
			MaxGrpcTimeout: &t,
		},
	}

	// classic CORS...
	cors := &route.CorsPolicy{}
	cors.AllowOrigin = []string{"*"}
	cors.AllowMethods = "GET, PUT, DELETE, POST, OPTIONS"
	cors.AllowHeaders = "keep-alive,user-agent,cache-control,content-type,content-transfer-encoding,custom-header-1,x-accept-content-transfer-encoding,x-accept-response-streaming,x-user-agent,x-grpc-web,grpc-timeout"
	cors.MaxAge = "1728000"
	cors.ExposeHeaders = "custom-header-1,grpc-status,grpc-message"
	cors.Enabled = &types.BoolValue{Value: true}

	v := route.VirtualHost{}
	v.Name = "local_service"
	v.Domains = []string{"*"}
	v.Routes = []route.Route{r}
	v.Cors = cors

	// the filters are the interesting part here
	// setups our handling of grpc-web and CORS
	manager := &hcm.HttpConnectionManager{
		CodecType:  hcm.AUTO,
		StatPrefix: "ingress_http",
		RouteSpecifier: &hcm.HttpConnectionManager_RouteConfig{
			RouteConfig: &api.RouteConfiguration{
				Name:         "local_route",
				VirtualHosts: []route.VirtualHost{v},
			},
		},
		HttpFilters: []*hcm.HttpFilter{{
			Name: util.GRPCWeb,
		},
			{
				Name: util.CORS,
			},
			{
				Name: util.Router,
			},
		},
	}

	pbst, err := util.MessageToStruct(manager)
	if err != nil {
		log.Println("error: ", err)
	}

	l.FilterChains = []listener.FilterChain{{
		Filters: []listener.Filter{{
			Name: util.HTTPConnectionManager,
			ConfigType: &listener.Filter_Config{
				Config: pbst,
			},
		}},
	}}

	return &l
}

func buildEndpoints(mgmtEndpoints []MgmtEndpoint) *api.ClusterLoadAssignment {
	// endpoints are members of the Envoy cluster that will respond to API calls
	// in our case, they are the API endpoints of the bcg-mgmt servers
	var endpoints []endpoint.LbEndpoint
	for _, mgmtEndpoint := range mgmtEndpoints {
		lbEndpoint := buildEndpoint(mgmtEndpoint)
		endpoints = append(endpoints, lbEndpoint)
	}

	la := &api.ClusterLoadAssignment{
		ClusterName: clusterName,
		Endpoints: []endpoint.LocalityLbEndpoints{{
			LbEndpoints: endpoints,
		}},
	}

	return la
}

func buildEndpoint(mgmtEndpoint MgmtEndpoint) endpoint.LbEndpoint {
	return endpoint.LbEndpoint{
		Endpoint: &endpoint.Endpoint{
			Address: &core.Address{
				Address: &core.Address_SocketAddress{
					SocketAddress: &core.SocketAddress{
						Protocol: core.TCP,
						Address:  mgmtEndpoint.Host,
						PortSpecifier: &core.SocketAddress_PortValue{
							PortValue: mgmtEndpoint.Port,
						},
					},
				},
			},
		},
	}
}
