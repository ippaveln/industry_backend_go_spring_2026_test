package main

import "fmt"

func main() {
	res, err := fizzBuzz(15)
	if err != nil {
		fmt.Println("Error:", err.Error())
	} else {
		fmt.Println(res) // Expected output: "FizzBuzz"
	}

	res, err = fizzBuzz(3)
	if err != nil {
		fmt.Println("Error:", err.Error())
	} else {
		fmt.Println(res) // Expected output: "Fizz"
	}

	res, err = fizzBuzz(5)
	if err != nil {
		fmt.Println("Error:", err.Error())
	} else {
		fmt.Println(res) // Expected output: "Buzz"
	}

	res, err = fizzBuzz(7)
	if err != nil {
		fmt.Println("Error:", err.Error())
	} else {
		fmt.Println(res) // Expected output: "7"
	}

	res, err = fizzBuzz(-5)
	if err != nil {
		fmt.Println("Error:", err.Error()) // Expected output: "Error: input must be a positive integer"
	} else {
		fmt.Println(res)
	}
}
