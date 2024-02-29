package main

import (
	"waku-poker-planning/cmd"
	"waku-poker-planning/config"
)

/*
    waku-pp connect 0x1234
	waku-pp new --name="six78 sprint 42" --fleet="wakuv2.prod"
*/

func main() {
	config.SetupLogger()
	cmd.Execute()
}
