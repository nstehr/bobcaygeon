package raop

import (
	"fmt"
	"net"
	"testing"
)

func TestResponseGenerate(t *testing.T) {
	expectedResp := "r89JJyLNRJ0RT/pI7OqyDzyF0ggoUY0BmpFB9hsIDkziT+TYZ6coZwdBX8AQWQiNGYQBSNzcFWQj41kGcUGOhE2OxnphwHjraZRvF5bwvcvjKEFmkJTtEDnfLvYB41MfzTbWDWA3PSXxVkOrfnMb0hRnS6Es4WWfuSzDDRKQBQUUvob4mrHh9QuMYU+uTbOEE8zXY4QWAjQuOJH8vPSyUmonJLRRdtftgMqxfRjPEJV+4XuZ5vv347ahg3Yr8K12kKJ7axyrJVbF6ghkkCM64Xn6iD6x7p453VjS5gtuz8pLECidA8yudBdJPIASAIRNownnuL/7GQy1bmRIFDvhsw"
	mac, _ := net.ParseMAC("54:52:00:b8:58:77")
	resp, _ := generateChallengeResponse("gY3cmhtK9LnECNUlXFb0qg==", mac, "192.168.0.15")
	if resp != expectedResp {
		t.Error(fmt.Sprintf("Expected: %s\r\n Got: %s", expectedResp, resp))
	}
}
