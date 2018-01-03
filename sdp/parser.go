package sdp

import (
	"bufio"
	"io"
	"strconv"
	"strings"
)

// Parse parses out an SDP packet into a SDP struct
func Parse(r io.Reader) (*SessionDescription, error) {
	sdp := NewSessionDescription()
	s := bufio.NewScanner(r)
	for s.Scan() {
		parts := strings.SplitN(s.Text(), "=", 2)
		typePart := parts[0]
		valuePart := parts[1]
		switch typePart {
		case "v":
			version, err := strconv.Atoi(valuePart)
			if err != nil {
				return nil, err
			}
			sdp.Version = version
		case "o":
			// <username> <sess-id> <sess-version> <nettype> <addrtype> <unicast-address>
			originParts := strings.Fields(valuePart)
			origin := Origin{}
			origin.Username = originParts[0]
			origin.SessionID = originParts[1]
			origin.SessionVersion = originParts[2]
			origin.NetType = originParts[3]
			origin.AddrType = originParts[4]
			origin.UnicastAddress = originParts[5]
			sdp.Origin = origin
		case "s":
			sdp.SessionName = valuePart
		case "c":
			// <nettype> <addrtype> <connection-address>
			connectionParts := strings.Fields(valuePart)
			connect := ConnectData{}
			connect.NetType = connectionParts[0]
			connect.AddrType = connectionParts[1]
			connect.ConnectionAddress = connectionParts[2]
			sdp.ConnectData = connect
		case "t":
			// <start-time> <stop-time>
			timingParts := strings.Fields(valuePart)
			timing := Timing{}
			start, err := strconv.Atoi(timingParts[0])
			if err != nil {
				return nil, err
			}
			stop, err := strconv.Atoi(timingParts[1])
			if err != nil {
				return nil, err
			}
			timing.StartTime = start
			timing.Stopime = stop
			sdp.Timing = timing
		case "m":
			// <media> <port>/<number of ports> <proto> <fmt>
			mediaParts := strings.Fields(valuePart)
			media := MediaDescription{}
			media.Media = mediaParts[0]
			media.Port = mediaParts[1]
			media.Proto = mediaParts[2]
			media.Fmt = mediaParts[3]
			sdp.MediaDescription = append(sdp.MediaDescription, media)
		case "a":
			attributeParts := strings.Split(valuePart, ":")
			sdp.Attributes[attributeParts[0]] = attributeParts[1]
		case "i":
			sdp.Information = valuePart
		}
		//TODO: handle all parameters

	}
	return sdp, nil
}
