package rtsp

import (
	"bytes"
	"fmt"
	"strings"
)

// Request RTSP request
type Request struct {
	Method     Method
	RequestURI string
	protocol   string
	Headers    map[string]string
	Body       string
}

// Response RTSP response
type Response struct {
	Headers  map[string]string
	Body     string
	Status   Status
	protocol string
}

func (r *Request) String() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("Protocol: %s\r\nMethod: %s\r\nRequest URI: %s\r\n", r.protocol, r.Method.String(), r.RequestURI))
	buffer.WriteString("Headers:\r\n")
	for k, v := range r.Headers {
		buffer.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	buffer.WriteString(fmt.Sprintf("Body:\r\n%s", r.Body))
	return buffer.String()
}

func (r *Response) String() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("Protocol: %s\r\nStatus: %s\r\n", r.protocol, r.Status.String()))
	buffer.WriteString("Headers:\r\n")
	for k, v := range r.Headers {
		buffer.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	buffer.WriteString(fmt.Sprintf("Body:\r\n%s", r.Body))
	return buffer.String()
}

var methods = map[string]Method{
	strings.ToLower(Describe.String()):      Describe,
	strings.ToLower(Announce.String()):      Announce,
	strings.ToLower(Get_Parameter.String()): Get_Parameter,
	strings.ToLower(Options.String()):       Options,
	strings.ToLower(Play.String()):          Play,
	strings.ToLower(Pause.String()):         Pause,
	strings.ToLower(Record.String()):        Record,
	strings.ToLower(Setup.String()):         Setup,
	strings.ToLower(Set_Parameter.String()): Set_Parameter,
	strings.ToLower(Teardown.String()):      Teardown,
	strings.ToLower(Flush.String()):         Flush,
}

// getMethod converts string to Method enum value, returning error if it can't map
func getMethod(method string) (Method, error) {
	m, exists := methods[strings.ToLower(method)]
	if !exists {
		return -1, fmt.Errorf("Not valid method: %s", method)
	}
	return m, nil
}

// GetMethods all RTSP methods as a slice of strings
func GetMethods() []string {
	keys := make([]string, 0, len(methods))
	for k := range methods {
		keys = append(keys, strings.ToUpper(k))
	}
	return keys
}
