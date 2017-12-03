package rtsp

import (
	"fmt"
	"log"
	"net"
)

type RequestHandler func(req *Request, resp *Response, localAddr string, remoteAddr string)

type RtspServer struct {
	port     int
	handlers map[Method]RequestHandler
	done     chan bool
}

func NewRtspServer(port int) *RtspServer {
	server := RtspServer{}
	server.port = port
	server.done = make(chan bool)
	server.handlers = make(map[Method]RequestHandler)
	return &server
}

func (r *RtspServer) AddHandler(m Method, rh RequestHandler) {
	r.handlers[m] = rh
}

// Stop stops the RTSP server
func (r *RtspServer) Stop() {
	log.Println("Stopping RTSP server")
	r.done <- true
}

// Start creates listening socket for the RTSP connection
func (r *RtspServer) Start(verbose bool) {
	log.Println(fmt.Sprintf("Starting RTSP server on port: %d", r.port))
	tcpListen, err := net.Listen("tcp", fmt.Sprintf(":%d", r.port))
	if err != nil {
		log.Fatal("Error listening:", err.Error())
	}

	defer tcpListen.Close()

	//handle TCP connections
	go func() {
		for {
			// Listen for an incoming connection.
			conn, err := tcpListen.Accept()
			if err != nil {
				log.Fatal("Error accepting: ", err.Error())
			}
			go read(conn, r.handlers, verbose)
		}
	}()

	<-r.done
}

func read(conn net.Conn, handlers map[Method]RequestHandler, verbose bool) {
	for {
		request, err := readRequest(conn)
		if err != nil {
			log.Println("Error reading data: ", err.Error())
			conn.Close()
			return
		}

		if verbose {
			log.Println("Received Request")
			log.Println(request.PrettyFormatted())
		}

		handler, exists := handlers[request.Method]
		if !exists {
			continue
		}
		resp := Response{Headers: make(map[string]string)}
		// for now we just stick in the protocol (protocol/version) from the request
		resp.protocol = request.protocol
		// same with CSeq
		resp.Headers["CSeq"] = request.Headers["CSeq"]
		// invokes the client specified handler to build the response
		localAddr := conn.LocalAddr().(*net.TCPAddr).IP
		remoteAddr := conn.RemoteAddr().(*net.TCPAddr).IP
		handler(request, &resp, localAddr.String(), remoteAddr.String())
		if verbose {
			log.Println("Outbound Response")
			log.Println(resp.PrettyFormatted())
		}
		writeResponse(conn, &resp)

	}
}
