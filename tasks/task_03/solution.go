package main

import (
	"errors"
	"fmt"
)

func fizzBuzz(n int) (string, error) {
	switch {
	case n <= 0:
		return "", errors.New("input must be a positive integer")
	case n%15 == 0:
		return "FizzBuzz", nil
	case n%3 == 0:
		return "Fizz", nil
	case n%5 == 0:
		return "Buzz", nil
	default:
		return fmt.Sprintf("%d", n), nil
	}
}
