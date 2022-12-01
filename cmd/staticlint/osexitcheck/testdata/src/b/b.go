package main

import "os"

func main() {
	a()
	Exit(1)
	os.Exit(1) // want "using os.Exit is not recommended"
}

func a() {
	os.Exit(1)
}

func Exit(i int) {

}
