package rtsp

import "testing"

func TestMethodExists(t *testing.T) {
	method, err := getMethod("options")
	if err != nil {
		t.Error("Expected nil err value", err)
	}
	if method != Options {
		t.Error("Expected Options, got: ", method)
	}
}

func TestMethodExistsCaseSensitive(t *testing.T) {
	method, err := getMethod("OPTIONS")
	if err != nil {
		t.Error("Expected nil err value", err)
	}
	if method != Options {
		t.Error("Expected Options, got: ", method)
	}
}

func TestMethodNotExists(t *testing.T) {
	_, err := getMethod("foo")
	if err == nil {
		t.Error("Expected non nil err value", err)
	}

}
