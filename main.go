package main

import "irnc"

func main() {
	irnc.Init()
	defer irnc.Finish()
	irnc.RunGUI()
}
