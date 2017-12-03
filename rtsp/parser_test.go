package rtsp

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestOptionsParse(t *testing.T) {
	options :=
		"OPTIONS * RTSP/1.0\r\n" +
			"CSeq: 1\r\n" +
			"User-Agent: iTunes/12.5.1 (Macintosh; OS X 10.11.6)\r\n" +
			"Client-Instance: 67F67C1CAA66A2F4\r\n" +
			"DACP-ID: 67F67C1CAA66A2F4\r\n" +
			"Active-Remote: 1721127963\r\n" +
			"\r\n"

	r := strings.NewReader(options)
	msg, err := readRequest(r)

	log.Println(msg)

	if err != nil {
		t.Error("Expected non nil err value", err)
	}
	if msg.Method != Options {
		t.Error("Expected OPTIONS got: ", msg.Method)
	}
	if msg.protocol != "RTSP/1.0" {
		t.Error("Expected RTSP/1.0 got: ", msg.protocol)
	}
	if len(msg.Headers) != 5 {
		t.Error("Unexpected amount of headers: ", len(msg.Headers))
	}
	// test a couple of the headers
	if msg.Headers["CSeq"] != "1" {
		t.Error("Unexpected CSeq", msg.Headers["CSeq"])
	}
	if msg.Headers["Client-Instance"] != "67F67C1CAA66A2F4" {
		t.Error("Unexpected Client-Instance", msg.Headers["Client-Instance"])
	}

}

func TestParseImproperRequestLine(t *testing.T) {
	options :=
		"OPTIONS *\r\n" +
			"CSeq: 1\r\n" +
			"User-Agent: iTunes/12.5.1 (Macintosh; OS X 10.11.6)\r\n" +
			"Client-Instance: 67F67C1CAA66A2F4\r\n" +
			"DACP-ID: 67F67C1CAA66A2F4\r\n" +
			"Active-Remote: 1721127963\r\n" +
			"\r\n"

	r := strings.NewReader(options)
	_, err := readRequest(r)
	if err == nil {
		t.Error("Expected error ")
	}

}

func TestParseImproperHeader(t *testing.T) {
	options :=
		"OPTIONS * RTSP/1.0\r\n" +
			"CSeq: 1\r\n" +
			"User-Agent\r\n" +
			"Client-Instance: 67F67C1CAA66A2F4\r\n" +
			"DACP-ID: 67F67C1CAA66A2F4\r\n" +
			"Active-Remote: 1721127963\r\n" +
			"\r\n"

	r := strings.NewReader(options)
	_, err := readRequest(r)
	if err == nil {
		t.Error("Expected non nil err value", err)
	}
}

func TestBuildResponse(t *testing.T) {
	respString :=
		"RTSP/1.0 200 Ok\r\n" +
			"CSeq: 1\r\n" +
			"Client-Instance: 67F67C1CAA66A2F4\r\n" +
			"\r\n"
	resp := Response{}
	headers := make(map[string]string)
	headers["CSeq"] = "1"
	headers["Client-Instance"] = "67F67C1CAA66A2F4"
	resp.protocol = "RTSP/1.0"
	resp.Headers = headers
	resp.Status = Ok
	var b bytes.Buffer
	n, err := writeResponse(&b, &resp)
	if err != nil {
		t.Error("Expected nil err value", err)
	}
	if n <= 0 {
		t.Error("No bytes written")
	}
	if respString != b.String() {
		t.Error("Non matching response generated. Expected:"+respString+"got:", b.String())
	}

}
