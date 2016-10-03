package main

import (
	"github.com/floeit/floe/floe"
	"github.com/floeit/floe/testfloe"
)

func main() {
	floe.Prep()
	floe.Start("Test", "Test floes", testfloe.GetFloes)
}
