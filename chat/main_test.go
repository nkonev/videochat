package main

import (
	"github.com/stretchr/testify/assert"
	test "net/http/httptest"
	"os"
	"testing"
)

// X-Auth-ExpiresIn
// 2020-03-17 08:36:04 +0000 UTC

// X-Auth-Username
// tester

// X-Auth-Userid
// tester

// X-Auth-UserId
// b01fb567-3f78-463b-8102-6da43b474705

func TestMain(m *testing.M) {
	setup()
	retCode := m.Run()
	shutdown()
	os.Exit(retCode)
}

func shutdown() {}

func setup() {}

func TestExtractAuth(t *testing.T) {
	req := test.NewRequest("GET", "/should-be-secured", nil)
	headers := map[string][]string{
		"X-Auth-Expiresin": {"1590022342295000"},
		"X-Auth-Username":  {"tester"},
		"X-Auth-Userid":    {"1"},
	}
	req.Header = headers

	auth, err := extractAuth(req)
	assert.Nil(t, err)
	assert.Equal(t, int64(1), auth.UserId)
	assert.Equal(t, "tester", auth.UserLogin)
	assert.Equal(t, int64(1590022342), auth.ExpiresAt)
}
