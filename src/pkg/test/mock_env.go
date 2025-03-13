package test

import "api.codprotect.app/src/pkg/global"

func SetupMockEnv() {
	global.RESEND_API_KEY = "re_jTE5NuYc_CGLrk8mHqP7p3NTBKkQQSaVe"
	global.RESEND_FROM = "LnLink No-Reply<lnlink@test-emails.cbt.bg>"
	global.MONGO_DB_URI = "mongodb://admin:admin@localhost:27017/api-lnlink-app?authSource=admin"
	global.MONGO_DB_NAME = "api-lnlink-app"
	global.MONGO_CLIENT = nil
}
