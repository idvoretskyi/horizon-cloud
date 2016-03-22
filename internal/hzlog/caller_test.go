package hzlog

import "testing"

func TestGetCallerInfo(t *testing.T) {
	info := getCallerInfo(1)
	if info.File != "hzlog/caller_test.go" ||
		info.Line <= 0 {
		t.Errorf("getCallerInfo(1) = %#v", info)
	}

	sameInfo := func() callerInfo {
		return getCallerInfo(2)
	}()
	if sameInfo.File != info.File {
		t.Errorf("wanted %#v but got %#v", info, sameInfo)
	}
}

func TestGetCallerInfoOutOfRange(t *testing.T) {
	info := getCallerInfo(9999)
	want := callerInfo{"???", 0}
	if info != want {
		t.Errorf("getCallerInfo(9999) = %#v, but wanted %#v", info, want)
	}
}
