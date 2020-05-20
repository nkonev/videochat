package main

import (
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"io"
	"net/http"
	test "net/http/httptest"
	"nkonev.name/chat/client"
	"nkonev.name/chat/handlers"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/utils"
	"os"
	"testing"
)

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

	auth, err := handlers.ExtractAuth(req)
	assert.Nil(t, err)
	assert.Equal(t, int64(1), auth.UserId)
	assert.Equal(t, "tester", auth.UserLogin)
	assert.Equal(t, int64(1590022342), auth.ExpiresAt)
}

func request(method, path string, body io.Reader, e *echo.Echo) (int, string, http.Header) {
	req := test.NewRequest(method, path, body)
	Header := map[string][]string{
		echo.HeaderContentType: {"application/json"},
		"X-Auth-Expiresin":     {"1590022342295000"},
		"X-Auth-Username":      {"tester"},
		"X-Auth-Userid":        {"1"},
	}
	req.Header = Header
	rec := test.NewRecorder()
	e.ServeHTTP(rec, req) // most wanted
	return rec.Code, rec.Body.String(), rec.HeaderMap
}

/*func runTest(container *fx.App, test func (e *echo.Echo)){
	//if migrationErr := container.Invoke(runMigration); migrationErr != nil {
	//	Logger.Panicf("Error during invoke migration: %v", migrationErr)
	//}

	if err := container.Invoke(func (e *echo.Echo){
		defer e.Close()

		test(e)
	}); err != nil {
		panic(err)
	}
}*/

func runTest(test func(e *echo.Echo)) *fx.App {
	configFile := utils.InitFlags("./config-dev/config.yml")
	utils.InitViper(configFile, "VIDEOCHAT")

	app := fx.New(
		fx.Logger(Logger),
		fx.Provide(
			client.NewRestClient,
			handlers.ConfigureCentrifuge,
			configureEcho,
			configureStaticMiddleware,
			handlers.ConfigureAuthMiddleware,
			configureDb,
		),
		fx.Invoke(
			runMigrations,
			//runCentrifuge,
			func(e *echo.Echo) {
				defer e.Close()
				test(e)
			},
		),
	)
	//app.Run()

	return app
}

func TestGetChats(t *testing.T) {
	//container := setUpContainerForIntegrationTests()

	runTest(func(e *echo.Echo) {
		c, b, _ := request("GET", "/chat", nil, e)
		assert.Equal(t, http.StatusOK, c)
		assert.NotEmpty(t, b)
	})
}
