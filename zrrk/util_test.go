package zrrk

import "testing"

func TestContainStrings(t *testing.T) {
	if !ContainStrings("abcd", "d") {
		t.Error("ContainStrings failed")
	}
	if !ContainStrings("abcd", "cd") {
		t.Error("ContainStrings failed")
	}
	if ContainStrings("abcd", "de") {
		t.Error("ContainStrings failed")
	}
	if !ContainStrings("abcd", "") {
		t.Error("ContainStrings failed")
	}
}
