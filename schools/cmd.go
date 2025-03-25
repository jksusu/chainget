package main

import "chainget/global"

func main() {
	global.InitConfig()
	//subSlot()

	NewItmClient().Run() //

	//FlashBotsClient{}.Run()
}
