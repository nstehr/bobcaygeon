const { GetNodesRequest, GetNodesResponse } = require('./management_pb.js');
const { BobcaygeonManagementClient } = require('./management_grpc_web_pb.js');

var mgmtService = new BobcaygeonManagementClient('http://localhost:9211');

var request = new GetNodesRequest();


mgmtService.getNodes(request, {}, function (err, response) {
    console.log(err);
    console.log(response.getNodesList());
});