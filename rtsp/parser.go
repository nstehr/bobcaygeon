package rtsp

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// https://tools.ietf.org/html/rfc2326#page-19
func readRequest(r io.Reader) (*Request, error) {

	req := new(Request)
	buf := bufio.NewReader(r)
	headers := make(map[string]string)

	// first line of the request will be the request line
	requestLine, err := buf.ReadString('\n')
	if err != nil {
		return nil, err
	}
	requestLine = strings.Trim(requestLine, "\r\n")
	requestLineParts := strings.Split(requestLine, " ")

	if len(requestLineParts) != 3 {
		return nil, fmt.Errorf("Improperly formatted request line: %s", requestLine)
	}

	method, err := getMethod(requestLineParts[0])

	if err != nil {
		return nil, fmt.Errorf("Method does exist in RTSP protocol: %s", requestLineParts[0])
	}

	req.Method = method
	req.RequestURI = requestLineParts[1]
	req.protocol = requestLineParts[2]

	// now we can read the headers.
	// we read a line until we hit the empty line
	// which indicates all the headers have been processed
	for {
		headerField, err := buf.ReadString('\n')
		if err != nil {
			return nil, err
		}
		headerField = strings.Trim(headerField, "\r\n")
		if strings.Trim(headerField, "\r\n") == "" {
			break
		}
		headerParts := strings.Split(headerField, ":")
		if len(headerParts) < 2 {
			return nil, fmt.Errorf("Inproper header: %s", headerField)
		}
		headers[strings.TrimSpace(headerParts[0])] = strings.TrimSpace(headerParts[1])
	}

	req.Headers = headers

	contentLength, hasBody := req.Headers["Content-Length"]
	if !hasBody {
		return req, nil
	}

	// now read the body
	length, _ := strconv.Atoi(contentLength)
	bodyBuf := make([]byte, length)
	// makes sure we read the full length of the content
	io.ReadFull(buf, bodyBuf)
	req.Body = bodyBuf

	return req, nil
}

func writeResponse(w io.Writer, resp *Response) (n int, err error) {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("%s %d %s\r\n", resp.protocol, resp.Status, resp.Status.String()))
	for header, value := range resp.Headers {
		buffer.WriteString(fmt.Sprintf("%s: %s\r\n", header, value))
	}
	buffer.WriteString("\r\n")

	if resp.Body != "" {
		resp.Headers["Content-Length"] = strconv.Itoa(len(resp.Body))
		buffer.WriteString(resp.Body)
	}
	return w.Write(buffer.Bytes())
}
