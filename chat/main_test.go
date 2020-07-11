package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/labstack/echo/v4"
	"github.com/oliveagle/jsonpath"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"io"
	"net/http"
	test "net/http/httptest"
	"nkonev.name/chat/client"
	"nkonev.name/chat/db"
	"nkonev.name/chat/handlers"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/notifications"
	name_nkonev_aaa "nkonev.name/chat/proto"
	"nkonev.name/chat/utils"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	setup()
	retCode := m.Run()
	shutdown()
	os.Exit(retCode)
}

func shutdown() {

}

const aaaEmuPort = "8061"

func setup() {
	configFile := utils.InitFlags("./config-dev/config.yml")
	utils.InitViper(configFile, "")

	viper.Set("aaa.url.base", "http://api.site.local:"+aaaEmuPort)

	d, err := db.ConfigureDb(nil)
	defer d.Close()
	if err != nil {
		Logger.Panicf("Error during getting db connection for test: %v", err)
	} else {
		d.RecreateDb()
	}
}

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

func requestWithHeader(method, path string, h http.Header, body io.Reader, e *echo.Echo) (int, string, http.Header) {
	req := test.NewRequest(method, path, body)
	req.Header = h
	rec := test.NewRecorder()
	e.ServeHTTP(rec, req) // most wanted
	return rec.Code, rec.Body.String(), rec.Result().Header
}

type ProtobufAaaEmu struct{}

func (receiver ProtobufAaaEmu) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(200)

	userResp := &name_nkonev_aaa.UsersResponse{}
	u1 := &name_nkonev_aaa.UserDto{
		Id:     16161,
		Login:  "testor_protobuf",
		Avatar: "http://image.jpg",
	}
	u2 := &name_nkonev_aaa.UserDto{
		Id:    16162,
		Login: "testor_protobuf2",
	}
	var users = []*name_nkonev_aaa.UserDto{u1, u2}
	userResp.Users = users
	out, err := proto.Marshal(userResp)
	if err != nil {
		Logger.Errorln("Failed to encode get users request:", err)
		return
	}

	resp.Write(out)
}

func startAaaEmu() *http.Server {
	s := &http.Server{
		Addr:    ":" + aaaEmuPort,
		Handler: ProtobufAaaEmu{},
	}

	go func() {
		Logger.Info(s.ListenAndServe())
	}()

	restClient := client.NewRestClient()

	for i := 1; i <= 30; i++ {
		_, err := restClient.GetUsers([]int64{0}, context.Background())
		if err != nil {
			Logger.Infof("Awaiting while emulator have been started")
			time.Sleep(time.Second * 1)
			continue
		} else {
			break
		}
	}
	restClient.CloseIdleConnections()

	return s
}

func request(method, path string, body io.Reader, e *echo.Echo) (int, string, http.Header) {
	Header := map[string][]string{
		echo.HeaderContentType: {"application/json"},
		"X-Auth-Expiresin":     {"1590022342295000"},
		"X-Auth-Username":      {"tester"},
		"X-Auth-Userid":        {"1"},
	}
	return requestWithHeader(method, path, Header, body, e)
}

func runTest(t *testing.T, testFunc interface{}) *fxtest.App {
	var s fx.Shutdowner
	app := fxtest.New(
		t,
		fx.Logger(Logger),
		fx.Populate(&s),
		fx.Provide(
			client.NewRestClient,
			handlers.ConfigureCentrifuge,
			handlers.CreateSanitizer,
			configureEcho,
			configureStaticMiddleware,
			handlers.ConfigureAuthMiddleware,
			db.ConfigureDb,
			notifications.NewNotifications,
		),
		fx.Invoke(
			runMigrations,
			runCentrifuge,
			//runEcho,
			initJaeger,
			testFunc,
		),
	)
	defer app.RequireStart().RequireStop()
	assert.NoError(t, s.Shutdown(), "error in app shutdown")
	return app
}

func TestGetChats(t *testing.T) {
	runTest(t, func(e *echo.Echo) {
		c, b, _ := request("GET", "/chat", nil, e)
		assert.Equal(t, http.StatusOK, c)
		assert.NotEmpty(t, b)
	})
}

func getJsonPathResult(t *testing.T, body string, jsonpath0 string) interface{} {
	var jsonData interface{}
	assert.Nil(t, json.Unmarshal([]byte(body), &jsonData))
	res, err := jsonpath.JsonPathLookup(jsonData, jsonpath0)
	assert.Nil(t, err)
	assert.NotEmpty(t, res)
	return res
}

func TestGetChatsPaginated(t *testing.T) {
	emu := startAaaEmu()
	defer emu.Close()
	runTest(t, func(e *echo.Echo) {
		c, b, _ := request("GET", "/chat?page=2&size=3", nil, e)
		assert.Equal(t, http.StatusOK, c)
		assert.NotEmpty(t, b)

		Logger.Infof("Body: %v", b)

		typedTes := getJsonPathResult(t, b, "$.data.name").([]interface{})

		assert.Equal(t, 3, len(typedTes))

		assert.Equal(t, "sit", typedTes[0])
		assert.Equal(t, "amet", typedTes[1])
		assert.Equal(t, "With collegues", typedTes[2])

		// also check get additional info froma aaa emu
		firstChatParticipantLogins := getJsonPathResult(t, b, "$.data[0].participants.login").([]interface{})
		assert.Equal(t, "testor_protobuf", firstChatParticipantLogins[0])
		assert.Equal(t, "testor_protobuf2", firstChatParticipantLogins[1])

		firstChatParticipantAvatars := getJsonPathResult(t, b, "$.data[0].participants.avatar").([]interface{})
		assert.Equal(t, "http://image.jpg", firstChatParticipantAvatars[0])
		assert.Equal(t, nil, firstChatParticipantAvatars[1])
	})
}

func TestChatValidation(t *testing.T) {
	runTest(t, func(e *echo.Echo, db db.DB) {
		c, b, _ := request("POST", "/chat", strings.NewReader(`{"name": ""}`), e)
		assert.Equal(t, http.StatusBadRequest, c)
		textString := interfaceToString(getJsonPathResult(t, b, "$.name").(interface{}))
		assert.Equal(t, "cannot be blank", textString)

		c2, b2, _ := request("POST", "/chat", strings.NewReader(``), e)
		assert.Equal(t, http.StatusBadRequest, c2)
		textString2 := interfaceToString(getJsonPathResult(t, b2, "$.name").(interface{}))
		assert.Equal(t, "cannot be blank", textString2)

		c3, b3, _ := request("PUT", "/chat", strings.NewReader(``), e)
		assert.Equal(t, http.StatusBadRequest, c3)
		textString30 := interfaceToString(getJsonPathResult(t, b3, "$.name").(interface{}))
		assert.Equal(t, "cannot be blank", textString30)
		textString31 := interfaceToString(getJsonPathResult(t, b3, "$.id").(interface{}))
		assert.Equal(t, "cannot be blank", textString31)
	})
}

func TestChatCrud(t *testing.T) {
	runTest(t, func(e *echo.Echo, db db.DB) {
		// test not found
		c30, _, _ := request("GET", "/chat/666", nil, e)
		assert.Equal(t, http.StatusNotFound, c30)

		chatsBefore, _ := db.CountChats()
		c, b, _ := request("POST", "/chat", strings.NewReader(`{"name": "Ultra new chat"}`), e)
		assert.Equal(t, http.StatusCreated, c)

		chatsAfterCreate, _ := db.CountChats()
		assert.Equal(t, chatsBefore+1, chatsAfterCreate)

		idInterface := getJsonPathResult(t, b, "$.id").(interface{})
		idString := interfaceToString(idInterface)
		id, _ := utils.ParseInt64(idString)
		assert.True(t, id > 0)

		c3, b3, _ := request("GET", "/chat/"+idString, nil, e)
		assert.Equal(t, http.StatusOK, c3)
		nameString := interfaceToString(getJsonPathResult(t, b3, "$.name").(interface{}))
		assert.Equal(t, "Ultra new chat", nameString)

		c2, _, _ := request("PUT", "/chat", strings.NewReader(`{ "id": `+idString+`, "name": "Mega ultra new chat"}`), e)
		assert.Equal(t, http.StatusAccepted, c2)
		row := db.QueryRow("SELECT title FROM chat WHERE id = $1", id)
		var newTitle string
		assert.Nil(t, row.Scan(&newTitle))
		assert.Equal(t, "Mega ultra new chat", newTitle)

		c1, _, _ := request("DELETE", "/chat/"+idString, nil, e)
		assert.Equal(t, http.StatusAccepted, c1)
		chatsAfterDelete, _ := db.CountChats()
		assert.Equal(t, chatsBefore, chatsAfterDelete)
	})
}

func interfaceToString(inter interface{}) string {
	return fmt.Sprintf("%v", inter)
}

func TestGetMessagesPaginated(t *testing.T) {
	runTest(t, func(e *echo.Echo) {
		c, b, _ := request("GET", "/chat/1/message?page=2&size=3", nil, e)
		assert.Equal(t, http.StatusOK, c)
		assert.NotEmpty(t, b)

		typedTes := getJsonPathResult(t, b, "$.text").([]interface{})

		assert.Equal(t, 3, len(typedTes))

		assert.True(t, strings.HasPrefix(interfaceToString(typedTes[0]), "generated_message5"))
		assert.True(t, strings.HasPrefix(interfaceToString(typedTes[1]), "generated_message6"))
		assert.True(t, strings.HasPrefix(interfaceToString(typedTes[2]), "generated_message7"))
	})
}

func TestMessageValidation(t *testing.T) {
	runTest(t, func(e *echo.Echo, db db.DB) {
		c, b, _ := request("POST", "/chat/1/message", strings.NewReader(`{"text": ""}`), e)
		assert.Equal(t, http.StatusBadRequest, c)
		textString := interfaceToString(getJsonPathResult(t, b, "$.text").(interface{}))
		assert.Equal(t, "cannot be blank", textString)

		c2, b2, _ := request("POST", "/chat/1/message", strings.NewReader(``), e)
		assert.Equal(t, http.StatusBadRequest, c2)
		textString2 := interfaceToString(getJsonPathResult(t, b2, "$.text").(interface{}))
		assert.Equal(t, "cannot be blank", textString2)
	})
}

func TestMessageCrud(t *testing.T) {
	runTest(t, func(e *echo.Echo, db db.DB) {
		messagesBefore, _ := db.CountMessages()
		c, b, _ := request("POST", "/chat/1/message", strings.NewReader(`{"text": "Ultra new message"}`), e)
		assert.Equal(t, http.StatusCreated, c)

		messagesAfterCreate, _ := db.CountMessages()
		assert.Equal(t, messagesBefore+1, messagesAfterCreate)

		idInterface := getJsonPathResult(t, b, "$.id").(interface{})
		idString := interfaceToString(idInterface)
		id, _ := utils.ParseInt64(idString)
		assert.True(t, id > 0)

		c3, b3, _ := request("GET", "/chat/1/message/"+idString, nil, e)
		assert.Equal(t, http.StatusOK, c3)
		textString := interfaceToString(getJsonPathResult(t, b3, "$.text").(interface{}))
		assert.Equal(t, "Ultra new message", textString)

		c4, _, _ := request("PUT", "/chat/1/message", strings.NewReader(`{"text": "Edited ultra new message", "id": `+idString+`}`), e)
		assert.Equal(t, http.StatusCreated, c4)

		c5, b5, _ := request("GET", "/chat/1/message/"+idString, nil, e)
		assert.Equal(t, http.StatusOK, c5)
		textString5 := interfaceToString(getJsonPathResult(t, b5, "$.text").(interface{}))
		assert.Equal(t, "Edited ultra new message", textString5)

		dateTimeInterface5 := interfaceToString(getJsonPathResult(t, b5, "$.editDateTime").(interface{}))
		assert.NotEmpty(t, dateTimeInterface5)

		c1, _, _ := request("DELETE", "/chat/1/message/"+idString, nil, e)
		assert.Equal(t, http.StatusAccepted, c1)
		messagesAfterDelete, _ := db.CountMessages()
		assert.Equal(t, messagesBefore, messagesAfterDelete)
	})
}

func TestMessageIsSanitized(t *testing.T) {
	runTest(t, func(e *echo.Echo, db db.DB) {
		c, b, _ := request("POST", "/chat/1/message", strings.NewReader(`{"text": "<a onblur=\"alert(secret)\" href=\"http://www.google.com\">Google</a>"}`), e)
		assert.Equal(t, http.StatusCreated, c)

		idInterface := getJsonPathResult(t, b, "$.id").(interface{})
		idString := interfaceToString(idInterface)

		c3, b3, _ := request("GET", "/chat/1/message/"+idString, nil, e)
		assert.Equal(t, http.StatusOK, c3)
		textInterface := getJsonPathResult(t, b3, "$.text").(interface{})
		textString := interfaceToString(textInterface)
		assert.Equal(t, `<a href="http://www.google.com" rel="nofollow">Google</a>`, textString)
	})
}

func TestItIsNotPossibleToWriteToForeignChat(t *testing.T) {
	h1 := map[string][]string{
		echo.HeaderContentType: {"application/json"},
		"X-Auth-Expiresin":     {"1590022342295000"},
		"X-Auth-Username":      {"tester"},
		"X-Auth-Userid":        {"1"},
	}
	h2 := map[string][]string{
		echo.HeaderContentType: {"application/json"},
		"X-Auth-Expiresin":     {"1590022342295000"},
		"X-Auth-Username":      {"tester2"},
		"X-Auth-Userid":        {"2"},
	}

	runTest(t, func(e *echo.Echo, db db.DB) {
		c, b, _ := requestWithHeader("POST", "/chat", h2, strings.NewReader(`{"name": "Chat of second user"}`), e)
		assert.Equal(t, http.StatusCreated, c)
		idInterface := getJsonPathResult(t, b, "$.id").(interface{})
		idString := interfaceToString(idInterface)

		// test not found
		c3, _, _ := requestWithHeader("GET", "/chat/"+idString+"/message/666", h2, nil, e)
		assert.Equal(t, http.StatusNotFound, c3)

		// first user tries to write to second user's chat
		c2, b2, _ := requestWithHeader("POST", "/chat/"+idString+"/message", h1, strings.NewReader(`{"text": "Ultra new message to the foreign chat"}`), e)
		assert.Equal(t, http.StatusBadRequest, c2)
		messageString := interfaceToString(getJsonPathResult(t, b2, "$.message").(interface{}))
		assert.Equal(t, "You are not allowed to write to this chat", messageString)
	})
}
