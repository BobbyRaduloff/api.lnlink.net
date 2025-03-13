package main

import (
	"context"

	"api.codprotect.app/src/pkg/global"
)

func main() {
	ctx := context.Background()

	global.Init()
	defer global.MONGO_CLIENT.Disconnect(ctx)

}
