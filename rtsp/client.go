package rtsp

import (
	"fmt"
	"net"
)

// Client Rtsp client
type Client struct {
	conn net.Conn
}

// NewClient instantiates a new client connecting to the address specified
func NewClient(address string, port int) (*Client, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", address, port))
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn}, nil
}

// Send will send a request to the server
func (c *Client) Send(request *Request) (*Response, error) {
	_, err := writeRequest(c.conn, request)
	if err != nil {
		return nil, err
	}
	resp, err := readResponse(c.conn)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
