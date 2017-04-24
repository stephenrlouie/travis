package main

import (
	"fmt"
)

func main() {
	err := A(0)
	if err != nil {
		fmt.Printf("A failed with ERROR: %v\n", err)
	}
	fmt.Printf("SUCCESS\n")
}

func A(a int) error {
	if a < 10 {
		return nil
	}
	return fmt.Errorf("Number larger than 10")
}
