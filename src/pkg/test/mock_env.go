package test

import "api.codprotect.app/src/pkg/global"

func SetupMockEnv() {
	global.RESEND_API_KEY = "re_YPdD92ig_GhLZ7epAB7ecMzMrorZPR2UZ"
	global.RESEND_FROM = "CoDProtect <no-reply@transactional.codprotect.app>"
	global.MONGO_DB_URI = "mongodb://admin:admin@localhost:27017/api.codprotect.app?authSource=admin"
	global.MONGO_DB_NAME = "api.codprotect.app"
	global.MONGO_CLIENT = nil

}
