package main

func greet(name string) string {
	// TODO: implement the function to return "Hello, <name>!"
	if name == "" {
		return "Hello, World!"
	}
	return "Hello, " + name + "!"
}
