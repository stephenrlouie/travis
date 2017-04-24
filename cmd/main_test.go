package main

import "testing"

func TestA(t *testing.T) {
	for i := 0; i < 20; i++ {
		err := A(i)
		if i < 10 && err != nil {
			t.Errorf("Test %d: Expected no error and received: %v", i, err)
		}
		if i >= 10 && err == nil {
			t.Errorf("Test %d: Expected an error and received nothing", i)
		}
	}
    t.Error("FORCED FAILURE")
}
