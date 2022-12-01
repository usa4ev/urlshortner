package main

import os1 "os"

func main() {
	a()
	Exit(1)
	os1.Exit(1) // want "using os.Exit is not recommended"
}

func a() {
	os1.Exit(1)
}

func Exit(i int) {

}
