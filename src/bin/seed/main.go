package main

import (
	"context"

	"api.lnlink.net/src/pkg/global"
	"api.lnlink.net/src/pkg/models/user"
)

func main() {
	ctx := context.Background()

	global.Init()
	defer global.MONGO_CLIENT.Disconnect(ctx)

	user.CreateUser(&user.UserAuth{
		Email:    "boris@lnlink.net",
		Password: "admin",
	})

}
