package email

import (
	"testing"

	"api.codprotect.app/src/pkg/test"

	"github.com/stretchr/testify/assert"
)

func TestEmailFail(t *testing.T) {
	test.SetupMockEnv()

	err := SendEmail("asdasdasdsad@daskjdalskjdlaksjdlas.com", "TEST", "<p>TEST</p>", "ASD")
	assert.Error(t, err, "This needs to fail instantly")
}

func TestEmailNotFail(t *testing.T) {
	test.SetupMockEnv()

	err := SendEmail("corporatebusinesstechnologies@gmail.com", "TEST", "<p>TEST</p>", "ASD")
	assert.Nil(t, err, "No error should come from here")
}
func TestNotEmail(t *testing.T) {
	test.SetupMockEnv()

	err := SendEmail("asd", "TEST", "<p>TEST</p>", "ASD")
	assert.Error(t, err, "This needs to fail instantly")
}
