package main

type os struct{}

func main() {
	os := os{}
	a(os)
	os.Exit(1)
	Exit(1)

}

func a(os os) {
	os.Exit(1)
}

func (o os) Exit(i int) {

}

func Exit(i int) {

}
