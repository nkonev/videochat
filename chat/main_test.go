package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/beliyav/go-amqp-reconnect/rabbitmq"
	"github.com/guregu/null"
	"github.com/labstack/echo/v4"
	"github.com/oliveagle/jsonpath"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"io"
	"io/ioutil"
	"net/http"
	test "net/http/httptest"
	"net/url"
	"nkonev.name/chat/client"
	"nkonev.name/chat/config"
	"nkonev.name/chat/db"
	"nkonev.name/chat/dto"
	"nkonev.name/chat/handlers"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/producer"
	myRabbitmq "nkonev.name/chat/rabbitmq"
	"nkonev.name/chat/services"
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

var userTester = base64.StdEncoding.EncodeToString([]byte("tester"))
var userTester2 = base64.StdEncoding.EncodeToString([]byte("tester2"))

func setup() {
	config.InitViper()

	viper.Set("aaa.url.base", "http://localhost:"+aaaEmuPort)

	d, err := db.ConfigureDb(nil)
	defer d.Close()
	if err != nil {
		Logger.Panicf("Error during getting db connection for test: %v", err)
	}
	d.RecreateDb()
}

func TestExtractAuth(t *testing.T) {
	req := test.NewRequest("GET", "/should-be-secured", nil)
	headers := map[string][]string{
		"X-Auth-Expiresin": {"1590022342295000"},
		"X-Auth-Username":  {userTester},
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

	u1 := &dto.User{
		Id:     1,
		Login:  "testor_protobuf",
		Avatar: null.StringFrom("http://image.jpg"),
	}
	u2 := &dto.User{
		Id:    2,
		Login: "testor_protobuf2",
	}
	var users = []*dto.User{u1, u2}
	out, err := json.Marshal(users)
	if err != nil {
		Logger.Errorln("Failed to encode get users request:", err)
		return
	}

	resp.Write(out)
}

func waitForAaaEmu() {
	restClient := client.NewRestClient()
	i := 0
	for ; i <= 30; i++ {
		_, err := restClient.GetUsers([]int64{0}, context.Background())
		if err != nil {
			Logger.Infof("Awaiting while emulator have been started")
			time.Sleep(time.Second * 1)
			continue
		} else {
			break
		}
	}
	if i == 30 {
		Logger.Panicf("Cannot await for aaa emu will be started")
	}
	Logger.Infof("Aaa emu have started")
	restClient.CloseIdleConnections()
}

// it's requires to call this method every time when we create real app with startAppFull()
func waitForChatServer() {
	restClient := client.NewRestClient()
	i := 0
	for ; i <= 30; i++ {
		expirationTime := utils.SecondsToStringMilliseconds(time.Now().Add(4 * time.Hour).Unix())
		contentType := "application/json;charset=UTF-8"
		requestHeaders1 := map[string][]string{
			"Accept":           {contentType},
			"Content-Type":     {contentType},
			"X-Auth-Expiresin": {expirationTime},
			"X-Auth-Username":  {userTester},
			"X-Auth-Userid":    {"1"},
		}
		getChatRequest := &http.Request{
			Method: "GET",
			Header: requestHeaders1,
			URL:    stringToUrl("http://localhost:1235/chat"),
		}
		getChatResponse, err := restClient.Do(getChatRequest)
		if err != nil {
			Logger.Infof("Awaiting while chat have been started - transport error")
			time.Sleep(time.Second * 1)
			continue
		} else if !(getChatResponse.StatusCode >= 200 && getChatResponse.StatusCode < 300) {
			Logger.Infof("Awaiting while chat have been started - non-2xx code")
			time.Sleep(time.Second * 1)
			continue
		} else {
			break
		}
	}
	if i == 30 {
		Logger.Panicf("Cannot await for chat will be started")
	}
	Logger.Infof("chat have started")
	restClient.CloseIdleConnections()
}

func startAaaEmu() *http.Server {
	s := &http.Server{
		Addr:    ":" + aaaEmuPort,
		Handler: ProtobufAaaEmu{},
	}

	go func() {
		Logger.Info(s.ListenAndServe())
	}()

	waitForAaaEmu()

	return s
}

func request(method, path string, body io.Reader, e *echo.Echo) (int, string, http.Header) {
	Header := map[string][]string{
		echo.HeaderContentType: {"application/json"},
		"X-Auth-Expiresin":     {"1590022342295000"},
		"X-Auth-Username":      {userTester},
		"X-Auth-Userid":        {"1"},
	}
	return requestWithHeader(method, path, Header, body, e)
}

func configureTestMigrations() *db.MigrationsConfig {
	return &db.MigrationsConfig{AppendTestData: true}
}

func runTest(t *testing.T, testFunc interface{}) *fxtest.App {
	var s fx.Shutdowner
	app := fxtest.New(
		t,
		fx.Logger(Logger),
		fx.Populate(&s),
		fx.Provide(
			configureTracer,
			client.NewRestClient,
			services.CreateSanitizer,
			services.CreateStripTags,
			services.StripStripSourcePolicy,
			handlers.NewChatHandler,
			handlers.NewMessageHandler,
			handlers.NewBlogHandler,
			configureEcho,
			handlers.ConfigureStaticMiddleware,
			handlers.ConfigureAuthMiddleware,
			configureTestMigrations,
			db.ConfigureDb,
			services.NewEvents,
			producer.NewRabbitEventsPublisher,
			producer.NewRabbitNotificationsPublisher,
			myRabbitmq.CreateRabbitMqConnection,
		),
		fx.Invoke(
			runMigrations,
			//runEcho,
			testFunc,
		),
	)
	defer app.RequireStart().RequireStop()
	assert.NoError(t, s.Shutdown(), "error in app shutdown")
	return app
}

func startAppFull(t *testing.T) (*fxtest.App, fx.Shutdowner) {
	var s fx.Shutdowner
	app := fxtest.New(
		t,
		fx.Logger(Logger),
		fx.Populate(&s),
		fx.Provide(
			configureTracer,
			client.NewRestClient,
			services.CreateSanitizer,
			services.CreateStripTags,
			services.StripStripSourcePolicy,
			handlers.NewChatHandler,
			handlers.NewMessageHandler,
			handlers.NewBlogHandler,
			configureEcho,
			handlers.ConfigureStaticMiddleware,
			handlers.ConfigureAuthMiddleware,
			configureTestMigrations,
			db.ConfigureDb,
			services.NewEvents,
			producer.NewRabbitEventsPublisher,
			producer.NewRabbitNotificationsPublisher,
			myRabbitmq.CreateRabbitMqConnection,
		),
		fx.Invoke(
			createFanoutNotificationsChannel,
			runMigrations,
			runEcho,
		),
	)
	waitForChatServer()
	return app, s
}

func createFanoutNotificationsChannel(connection *rabbitmq.Connection, lc fx.Lifecycle) error {
	consumeCh, err := connection.Channel(nil)
	if err != nil {
		return err
	}

	err = consumeCh.ExchangeDeclare(producer.EventsFanoutExchange, "fanout", true, false, false, false, nil)
	if err != nil {
		return err
	}
	return nil
}

func TestGetChats(t *testing.T) {
	runTest(t, func(e *echo.Echo) {
		c, b, _ := request("GET", "/chat", nil, e)
		assert.Equal(t, http.StatusOK, c)
		assert.NotEmpty(t, b)
	})
}

func getJsonPathResult(t *testing.T, body string, jsonpath0 string) interface{} {
	res := getJsonPathRaw(t, body, jsonpath0)
	assert.NotEmpty(t, res)
	return res
}

func getJsonPathRaw(t *testing.T, body string, jsonpath0 string) interface{} {
	var jsonData interface{}
	assert.Nil(t, json.Unmarshal([]byte(body), &jsonData))
	res, err := jsonpath.JsonPathLookup(jsonData, jsonpath0)
	assert.Nil(t, err)
	return res
}

func TestGetChatsPaginated(t *testing.T) {
	emu := startAaaEmu()
	defer emu.Close()
	runTest(t, func(e *echo.Echo) {
		// get initial page
		httpFirstPage, bodyFirstPage, _ := request("GET", "/chat", nil, e)
		assert.Equal(t, http.StatusOK, httpFirstPage)
		assert.NotEmpty(t, bodyFirstPage)
		typedResFirstPage := getJsonPathResult(t, bodyFirstPage, "$.data.name").([]interface{})

		assert.Equal(t, 40, len(typedResFirstPage))

		assert.Equal(t, "generated_chat1000", typedResFirstPage[0])
		assert.Equal(t, "generated_chat999", typedResFirstPage[1])
		assert.Equal(t, "generated_chat998", typedResFirstPage[2])
		assert.Equal(t, "generated_chat961", typedResFirstPage[39])

		// also check get additional info from aaa emu
		firstChatParticipantLogins := getJsonPathResult(t, bodyFirstPage, "$.data[0].participants.login").([]interface{})
		assert.Equal(t, "testor_protobuf", firstChatParticipantLogins[0])

		firstChatParticipantAvatars := getJsonPathResult(t, bodyFirstPage, "$.data[0].participants.avatar").([]interface{})
		assert.Equal(t, "http://image.jpg", firstChatParticipantAvatars[0])

		paginationToken := getJsonPathResult(t, bodyFirstPage, "$.paginationToken").(string)

		// get second page
		httpSecondPage, bodySecondPage, _ := request("GET", "/chat?paginationToken="+paginationToken, nil, e)
		assert.Equal(t, http.StatusOK, httpSecondPage)
		assert.NotEmpty(t, bodySecondPage)
		typedResSecondPage := getJsonPathResult(t, bodySecondPage, "$.data.name").([]interface{})

		assert.Equal(t, 40, len(typedResSecondPage))

		assert.Equal(t, "generated_chat960", typedResSecondPage[0])
		assert.Equal(t, "generated_chat959", typedResSecondPage[1])
		assert.Equal(t, "generated_chat958", typedResSecondPage[2])
		assert.Equal(t, "generated_chat921", typedResSecondPage[39])
	})
}

func TestGetChatsPaginatedSearch(t *testing.T) {
	runTest(t, func(e *echo.Echo) {
		// get initial page
		httpFirstPage, bodyFirstPage, _ := request("GET", "/chat?searchString=gen", nil, e)
		assert.Equal(t, http.StatusOK, httpFirstPage)
		assert.NotEmpty(t, bodyFirstPage)
		typedResFirstPage := getJsonPathResult(t, bodyFirstPage, "$.data.name").([]interface{})

		assert.Equal(t, 40, len(typedResFirstPage))

		assert.Equal(t, "generated_chat1000", typedResFirstPage[0])
		assert.Equal(t, "generated_chat999", typedResFirstPage[1])
		assert.Equal(t, "generated_chat998", typedResFirstPage[2])
		assert.Equal(t, "generated_chat961", typedResFirstPage[39])

		paginationToken := getJsonPathResult(t, bodyFirstPage, "$.paginationToken").(string)

		// get second page
		httpSecondPage, bodySecondPage, _ := request("GET", "/chat?searchString=gen&paginationToken="+paginationToken, nil, e)
		assert.Equal(t, http.StatusOK, httpSecondPage)
		assert.NotEmpty(t, bodySecondPage)
		typedResSecondPage := getJsonPathResult(t, bodySecondPage, "$.data.name").([]interface{})

		assert.Equal(t, 40, len(typedResSecondPage))

		assert.Equal(t, "generated_chat960", typedResSecondPage[0])
		assert.Equal(t, "generated_chat959", typedResSecondPage[1])
		assert.Equal(t, "generated_chat958", typedResSecondPage[2])
		assert.Equal(t, "generated_chat921", typedResSecondPage[39])
	})
}

func TestChatValidation(t *testing.T) {
	runTest(t, func(e *echo.Echo, db *db.DB) {
		c, b, _ := request("POST", "/chat", strings.NewReader(`{"name": ""}`), e)
		assert.Equal(t, http.StatusBadRequest, c)
		textString := utils.InterfaceToString(getJsonPathResult(t, b, "$.name").(interface{}))
		assert.Equal(t, "cannot be blank", textString)

		c2, b2, _ := request("POST", "/chat", strings.NewReader(``), e)
		assert.Equal(t, http.StatusBadRequest, c2)
		textString2 := utils.InterfaceToString(getJsonPathResult(t, b2, "$.name").(interface{}))
		assert.Equal(t, "cannot be blank", textString2)

		c3, b3, _ := request("PUT", "/chat", strings.NewReader(``), e)
		assert.Equal(t, http.StatusBadRequest, c3)
		textString30 := utils.InterfaceToString(getJsonPathResult(t, b3, "$.name").(interface{}))
		assert.Equal(t, "cannot be blank", textString30)
		textString31 := utils.InterfaceToString(getJsonPathResult(t, b3, "$.id").(interface{}))
		assert.Equal(t, "cannot be blank", textString31)
	})
}

func TestChatCrud(t *testing.T) {
	runTest(t, func(e *echo.Echo, db *db.DB) {
		// test not found
		c30, _, _ := request("GET", "/chat/50666", nil, e)
		assert.Equal(t, http.StatusNoContent, c30)

		chatsBefore, _ := db.CountChats()
		c, b, _ := request("POST", "/chat", strings.NewReader(`{"name": "Ultra new chat"}`), e)
		assert.Equal(t, http.StatusCreated, c)

		chatsAfterCreate, _ := db.CountChats()
		assert.Equal(t, chatsBefore+1, chatsAfterCreate)

		idInterface := getJsonPathResult(t, b, "$.id").(interface{})
		idString := utils.InterfaceToString(idInterface)
		id, _ := utils.ParseInt64(idString)
		assert.True(t, id > 0)

		c3, b3, _ := request("GET", "/chat/"+idString, nil, e)
		assert.Equal(t, http.StatusOK, c3)
		nameString := utils.InterfaceToString(getJsonPathResult(t, b3, "$.name").(interface{}))
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

func stringToUrl(s string) *url.URL {
	u, _ := url.Parse(s)
	return u
}

func stringToReadCloser(s string) io.ReadCloser {
	r := strings.NewReader(s)
	rc := ioutil.NopCloser(r)
	return rc
}

func TestCreateNewMessageMakesNotificationToOtherParticipant(t *testing.T) {
	app, s := startAppFull(t)
	defer app.RequireStart().RequireStop()

	emu := startAaaEmu()
	defer emu.Close()

	expirationTime := utils.SecondsToStringMilliseconds(time.Now().Add(2 * time.Hour).Unix())
	contentType := "application/json;charset=UTF-8"
	requestHeaders1 := map[string][]string{
		"Accept":           {contentType},
		"Content-Type":     {contentType},
		"X-Auth-Expiresin": {expirationTime},
		"X-Auth-Username":  {userTester},
		"X-Auth-Userid":    {"1"},
	}

	createChatRequest := &http.Request{
		Method: "POST",
		Header: requestHeaders1,
		Body:   stringToReadCloser(`{"name": "Chat for test the Centrifuge notifications about unread messages", "participantIds": [1, 2]}`),
		URL:    stringToUrl("http://localhost:1235/chat"),
	}

	cl := client.NewRestClient()
	createChatResponse, err := cl.Do(createChatRequest)
	assert.Nil(t, err)
	assert.Equal(t, 201, createChatResponse.StatusCode)

	var responseDto *dto.ChatDto = new(dto.ChatDto)
	body, err := ioutil.ReadAll(createChatResponse.Body)
	assert.Nil(t, err)
	err = json.Unmarshal(body, responseDto)
	assert.Nil(t, err)

	var chatIdString = fmt.Sprintf("%v", responseDto.Id)

	// send message to chat
	messageRequest := &http.Request{
		Method: "POST",
		Header: requestHeaders1,
		Body:   stringToReadCloser(`{"text": "Hello dude"}`),
		URL:    stringToUrl("http://localhost:1235/chat/" + chatIdString + "/message"),
	}
	messageResponse, err := cl.Do(messageRequest)
	assert.Nil(t, err)
	assert.Equal(t, 201, messageResponse.StatusCode)

	// send message to chat again
	messageRequest2 := &http.Request{
		Method: "POST",
		Header: requestHeaders1,
		Body:   stringToReadCloser(`{"text": "Hello dude"}`),
		URL:    stringToUrl("http://localhost:1235/chat/" + chatIdString + "/message"),
	}
	messageResponse2, err := cl.Do(messageRequest2)
	assert.Nil(t, err)
	assert.Equal(t, 201, messageResponse2.StatusCode)

	assert.NoError(t, s.Shutdown(), "error in app shutdown")
}

func TestBadRequestShouldReturn400(t *testing.T) {
	app, s := startAppFull(t)
	defer app.RequireStart().RequireStop()

	emu := startAaaEmu()
	defer emu.Close()

	expirationTime := utils.SecondsToStringMilliseconds(time.Now().Add(2 * time.Hour).Unix())
	contentType := "application/json;charset=UTF-8"
	requestHeaders1 := map[string][]string{
		"Accept":           {contentType},
		"Content-Type":     {contentType},
		"X-Auth-Expiresin": {expirationTime},
		"X-Auth-Username":  {userTester},
		"X-Auth-Userid":    {"1"},
	}

	createChatRequest := &http.Request{
		Method: "POST",
		Header: requestHeaders1,
		Body:   stringToReadCloser(`{"name": "Chat for test the Centrifuge notifications about unread messages", "participantIds": [1, 2]`),
		URL:    stringToUrl("http://localhost:1235/chat"),
	}

	cl := client.NewRestClient()
	createChatResponse, err := cl.Do(createChatRequest)
	assert.Nil(t, err)
	assert.Equal(t, 400, createChatResponse.StatusCode)

	assert.NoError(t, s.Shutdown(), "error in app shutdown")
}

func TestGetMessagesPaginated(t *testing.T) {
	runTest(t, func(e *echo.Echo) {
		// get first page
		httpFirstPage, bodyFirstPage, _ := request("GET", "/chat/1/message?startingFromItemId=6&size=3", nil, e)
		assert.Equal(t, http.StatusOK, httpFirstPage)
		assert.NotEmpty(t, bodyFirstPage)

		firstPageResult := []dto.DisplayMessageDto{}
		err := json.Unmarshal([]byte(bodyFirstPage), &firstPageResult)
		assert.NoError(t, err)

		assert.Equal(t, 3, len(firstPageResult))
		assert.True(t, strings.HasPrefix(firstPageResult[0].Text, "generated_message5"))
		assert.True(t, strings.HasPrefix(firstPageResult[1].Text, "generated_message6"))
		assert.True(t, strings.HasPrefix(firstPageResult[2].Text, "generated_message7"))
		assert.Equal(t, int64(7), firstPageResult[0].Id)
		assert.Equal(t, int64(8), firstPageResult[1].Id)
		assert.Equal(t, int64(9), firstPageResult[2].Id)


		// get second page
		httpSecondPage, bodySecondPage, _ := request("GET", "/chat/1/message?startingFromItemId=9&size=3", nil, e)
		assert.Equal(t, http.StatusOK, httpSecondPage)
		assert.NotEmpty(t, bodySecondPage)

		secondPageResult := []dto.DisplayMessageDto{}
		err = json.Unmarshal([]byte(bodySecondPage), &secondPageResult)
		assert.NoError(t, err)

		assert.Equal(t, 3, len(secondPageResult))
		assert.True(t, strings.HasPrefix(secondPageResult[0].Text, "generated_message8"))
		assert.True(t, strings.HasPrefix(secondPageResult[1].Text, "generated_message9"))
		assert.True(t, strings.HasPrefix(secondPageResult[2].Text, "generated_message10"))
		assert.Equal(t, int64(10), secondPageResult[0].Id)
		assert.Equal(t, int64(11), secondPageResult[1].Id)
		assert.Equal(t, int64(12), secondPageResult[2].Id)
	})
}

func TestGetMessagesPaginatedSearch(t *testing.T) {
	runTest(t, func(e *echo.Echo) {
		// get first page
		httpFirstPage, bodyFirstPage, _ := request("GET", "/chat/1/message?startingFromItemId=6&size=3&searchString=gen", nil, e)
		assert.Equal(t, http.StatusOK, httpFirstPage)
		assert.NotEmpty(t, bodyFirstPage)

		firstPageResult := []dto.DisplayMessageDto{}
		err := json.Unmarshal([]byte(bodyFirstPage), &firstPageResult)
		assert.NoError(t, err)

		assert.Equal(t, 3, len(firstPageResult))
		assert.True(t, strings.HasPrefix(firstPageResult[0].Text, "generated_message5"))
		assert.True(t, strings.HasPrefix(firstPageResult[1].Text, "generated_message6"))
		assert.True(t, strings.HasPrefix(firstPageResult[2].Text, "generated_message7"))
		assert.Equal(t, int64(7), firstPageResult[0].Id)
		assert.Equal(t, int64(8), firstPageResult[1].Id)
		assert.Equal(t, int64(9), firstPageResult[2].Id)


		// get second page
		httpSecondPage, bodySecondPage, _ := request("GET", "/chat/1/message?startingFromItemId=9&size=3", nil, e)
		assert.Equal(t, http.StatusOK, httpSecondPage)
		assert.NotEmpty(t, bodySecondPage)

		secondPageResult := []dto.DisplayMessageDto{}
		err = json.Unmarshal([]byte(bodySecondPage), &secondPageResult)
		assert.NoError(t, err)

		assert.Equal(t, 3, len(secondPageResult))
		assert.True(t, strings.HasPrefix(secondPageResult[0].Text, "generated_message8"))
		assert.True(t, strings.HasPrefix(secondPageResult[1].Text, "generated_message9"))
		assert.True(t, strings.HasPrefix(secondPageResult[2].Text, "generated_message10"))
		assert.Equal(t, int64(10), secondPageResult[0].Id)
		assert.Equal(t, int64(11), secondPageResult[1].Id)
		assert.Equal(t, int64(12), secondPageResult[2].Id)
	})
}

func TestGetMessagesHasHash(t *testing.T) {
	runTest(t, func(e *echo.Echo) {
		// get first page
		httpFirstPage, bodyFirstPage, _ := request("GET", "/chat/1/message?startingFromItemId=7&size=10&hasHash=true", nil, e)
		assert.Equal(t, http.StatusOK, httpFirstPage)
		assert.NotEmpty(t, bodyFirstPage)

		firstPageResult := []dto.DisplayMessageDto{}
		err := json.Unmarshal([]byte(bodyFirstPage), &firstPageResult)
		assert.NoError(t, err)

		assert.Equal(t, 10, len(firstPageResult))
		assert.True(t, strings.HasPrefix(firstPageResult[0].Text, "generated_message1"))
		assert.True(t, strings.HasPrefix(firstPageResult[1].Text, "generated_message2"))
		assert.True(t, strings.HasPrefix(firstPageResult[2].Text, "generated_message3"))
		assert.True(t, strings.HasPrefix(firstPageResult[3].Text, "generated_message4"))
		assert.True(t, strings.HasPrefix(firstPageResult[4].Text, "generated_message5"))
		assert.True(t, strings.HasPrefix(firstPageResult[5].Text, "generated_message6"))
		assert.True(t, strings.HasPrefix(firstPageResult[6].Text, "generated_message7"))
		assert.True(t, strings.HasPrefix(firstPageResult[7].Text, "generated_message8"))
		assert.True(t, strings.HasPrefix(firstPageResult[8].Text, "generated_message9"))
		assert.True(t, strings.HasPrefix(firstPageResult[9].Text, "generated_message10"))
		assert.Equal(t, int64(3), firstPageResult[0].Id)
		assert.Equal(t, int64(4), firstPageResult[1].Id)
		assert.Equal(t, int64(5), firstPageResult[2].Id)
		assert.Equal(t, int64(6), firstPageResult[3].Id)
		assert.Equal(t, int64(7), firstPageResult[4].Id)
		assert.Equal(t, int64(8), firstPageResult[5].Id)
		assert.Equal(t, int64(9), firstPageResult[6].Id)
		assert.Equal(t, int64(10), firstPageResult[7].Id)
		assert.Equal(t, int64(11), firstPageResult[8].Id)
		assert.Equal(t, int64(12), firstPageResult[9].Id)
	})
}

func TestGetMessagesHasHashSearch(t *testing.T) {
	runTest(t, func(e *echo.Echo) {
		// get first page
		httpFirstPage, bodyFirstPage, _ := request("GET", "/chat/1/message?startingFromItemId=7&size=10&hasHash=true&searchString=gen", nil, e)
		assert.Equal(t, http.StatusOK, httpFirstPage)
		assert.NotEmpty(t, bodyFirstPage)

		firstPageResult := []dto.DisplayMessageDto{}
		err := json.Unmarshal([]byte(bodyFirstPage), &firstPageResult)
		assert.NoError(t, err)

		assert.Equal(t, 10, len(firstPageResult))
		assert.True(t, strings.HasPrefix(firstPageResult[0].Text, "generated_message1"))
		assert.True(t, strings.HasPrefix(firstPageResult[1].Text, "generated_message2"))
		assert.True(t, strings.HasPrefix(firstPageResult[2].Text, "generated_message3"))
		assert.True(t, strings.HasPrefix(firstPageResult[3].Text, "generated_message4"))
		assert.True(t, strings.HasPrefix(firstPageResult[4].Text, "generated_message5"))
		assert.True(t, strings.HasPrefix(firstPageResult[5].Text, "generated_message6"))
		assert.True(t, strings.HasPrefix(firstPageResult[6].Text, "generated_message7"))
		assert.True(t, strings.HasPrefix(firstPageResult[7].Text, "generated_message8"))
		assert.True(t, strings.HasPrefix(firstPageResult[8].Text, "generated_message9"))
		assert.True(t, strings.HasPrefix(firstPageResult[9].Text, "generated_message10"))
		assert.Equal(t, int64(3), firstPageResult[0].Id)
		assert.Equal(t, int64(4), firstPageResult[1].Id)
		assert.Equal(t, int64(5), firstPageResult[2].Id)
		assert.Equal(t, int64(6), firstPageResult[3].Id)
		assert.Equal(t, int64(7), firstPageResult[4].Id)
		assert.Equal(t, int64(8), firstPageResult[5].Id)
		assert.Equal(t, int64(9), firstPageResult[6].Id)
		assert.Equal(t, int64(10), firstPageResult[7].Id)
		assert.Equal(t, int64(11), firstPageResult[8].Id)
		assert.Equal(t, int64(12), firstPageResult[9].Id)
	})
}

func TestMessageValidation(t *testing.T) {
	runTest(t, func(e *echo.Echo, db *db.DB) {
		c, b, _ := request("POST", "/chat/1/message", strings.NewReader(`{"text": ""}`), e)
		assert.Equal(t, http.StatusBadRequest, c)
		textString := utils.InterfaceToString(getJsonPathResult(t, b, "$.text").(interface{}))
		assert.Equal(t, "cannot be blank", textString)

		c2, b2, _ := request("POST", "/chat/1/message", strings.NewReader(``), e)
		assert.Equal(t, http.StatusBadRequest, c2)
		textString2 := utils.InterfaceToString(getJsonPathResult(t, b2, "$.text").(interface{}))
		assert.Equal(t, "cannot be blank", textString2)
	})
}

func TestMessageCrud(t *testing.T) {
	runTest(t, func(e *echo.Echo, db *db.DB) {
		messagesBefore, _ := db.CountMessages()
		c, b, _ := request("POST", "/chat/1/message", strings.NewReader(`{"text": "Ultra new message"}`), e)
		assert.Equal(t, http.StatusCreated, c)

		messagesAfterCreate, _ := db.CountMessages()
		assert.Equal(t, messagesBefore+1, messagesAfterCreate)

		idInterface := getJsonPathResult(t, b, "$.id").(interface{})
		idString := utils.InterfaceToString(idInterface)
		id, _ := utils.ParseInt64(idString)
		assert.True(t, id > 0)

		c3, b3, _ := request("GET", "/chat/1/message/"+idString, nil, e)
		assert.Equal(t, http.StatusOK, c3)
		textString := utils.InterfaceToString(getJsonPathResult(t, b3, "$.text").(interface{}))
		assert.Equal(t, "Ultra new message", textString)

		c4, _, _ := request("PUT", "/chat/1/message", strings.NewReader(`{"text": "Edited ultra new message", "id": `+idString+`}`), e)
		assert.Equal(t, http.StatusCreated, c4)

		c5, b5, _ := request("GET", "/chat/1/message/"+idString, nil, e)
		assert.Equal(t, http.StatusOK, c5)
		textString5 := utils.InterfaceToString(getJsonPathResult(t, b5, "$.text").(interface{}))
		assert.Equal(t, "Edited ultra new message", textString5)

		dateTimeInterface5 := utils.InterfaceToString(getJsonPathResult(t, b5, "$.editDateTime").(interface{}))
		assert.NotEmpty(t, dateTimeInterface5)

		c1, _, _ := request("DELETE", "/chat/1/message/"+idString, nil, e)
		assert.Equal(t, http.StatusAccepted, c1)
		messagesAfterDelete, _ := db.CountMessages()
		assert.Equal(t, messagesBefore, messagesAfterDelete)
	})
}

func TestMessageIsSanitized(t *testing.T) {
	runTest(t, func(e *echo.Echo, db *db.DB) {
		c, b, _ := request("POST", "/chat/1/message", strings.NewReader(`{"text": "<a onblur=\"alert(secret)\" href=\"http://www.google.com\">Google</a>"}`), e)
		assert.Equal(t, http.StatusCreated, c)

		idInterface := getJsonPathResult(t, b, "$.id").(interface{})
		idString := utils.InterfaceToString(idInterface)

		c3, b3, _ := request("GET", "/chat/1/message/"+idString, nil, e)
		assert.Equal(t, http.StatusOK, c3)
		textInterface := getJsonPathResult(t, b3, "$.text").(interface{})
		textString := utils.InterfaceToString(textInterface)
		assert.Equal(t, `<a href="http://www.google.com" rel="nofollow">Google</a>`, textString)
	})
}

func TestNotPossibleToWriteAMessageWithNotAllowedMediaUrl(t *testing.T) {
	runTest(t, func(e *echo.Echo, db *db.DB) {
		c, b, _ := request("POST", "/chat/1/message", strings.NewReader(`{"text": "<img src=\"http://malicious.example.com/virus.jpg\"> Lorem ipsum"}`), e)
		assert.Equal(t, http.StatusBadRequest, c)

		messageInterface := getJsonPathResult(t, b, "$.message").(interface{})
		messageString := utils.InterfaceToString(messageInterface)
		assert.Equal(t, "Media url is not allowed in image src: http://malicious.example.com/virus.jpg", messageString)
	})
}

func TestNotPossibleToEditAMessageAndSetNotAllowedMediaUrl(t *testing.T) {
	runTest(t, func(e *echo.Echo, db *db.DB) {
		c1, b1, _ := request("POST", "/chat/1/message", strings.NewReader(`{"text": "Lorem ipsum"}`), e)
		assert.Equal(t, http.StatusCreated, c1)
		idInterface := getJsonPathResult(t, b1, "$.id").(interface{})
		idString := utils.InterfaceToString(idInterface)

		c2, b2, _ := request("PUT", "/chat/1/message", strings.NewReader(fmt.Sprintf(`{ "id": %v, "text": "<img src=\"http://malicious.example.com/virus.jpg\"> Lorem ipsum"}`, idString)), e)
		assert.Equal(t, http.StatusBadRequest, c2)

		messageInterface := getJsonPathResult(t, b2, "$.message").(interface{})
		messageString := utils.InterfaceToString(messageInterface)
		assert.Equal(t, "Media url is not allowed in image src: http://malicious.example.com/virus.jpg", messageString)
	})
}

func TestItIsNotPossibleToWriteToForeignChat(t *testing.T) {
	h1 := map[string][]string{
		echo.HeaderContentType: {"application/json"},
		"X-Auth-Expiresin":     {"1590022342295000"},
		"X-Auth-Username":      {userTester}, // tester
		"X-Auth-Userid":        {"1"},
	}
	h2 := map[string][]string{
		echo.HeaderContentType: {"application/json"},
		"X-Auth-Expiresin":     {"1590022342295000"},
		"X-Auth-Username":      {userTester2}, // tester2
		"X-Auth-Userid":        {"2"},
	}

	runTest(t, func(e *echo.Echo, db *db.DB) {
		c, b, _ := requestWithHeader("POST", "/chat", h2, strings.NewReader(`{"name": "Chat of second user"}`), e)
		assert.Equal(t, http.StatusCreated, c)
		idInterface := getJsonPathResult(t, b, "$.id").(interface{})
		idString := utils.InterfaceToString(idInterface)

		// test not found
		c3, _, _ := requestWithHeader("GET", "/chat/"+idString+"/message/666", h2, nil, e)
		assert.Equal(t, http.StatusNotFound, c3)

		// first user tries to write to second user's chat
		c2, b2, _ := requestWithHeader("POST", "/chat/"+idString+"/message", h1, strings.NewReader(`{"text": "Ultra new message to the foreign chat"}`), e)
		assert.Equal(t, http.StatusBadRequest, c2)
		messageString := utils.InterfaceToString(getJsonPathResult(t, b2, "$.message").(interface{}))
		assert.Equal(t, "You are not allowed to write to this chat", messageString)
	})
}

func TestGetBlogsPaginated(t *testing.T) {
	runTest(t, func(e *echo.Echo) {
		// get first page
		httpFirstPage, bodyFirstPage, _ := request("GET", "/blog?startingFromItemId=6&size=3", nil, e)
		assert.Equal(t, http.StatusOK, httpFirstPage)
		assert.NotEmpty(t, bodyFirstPage)

		firstPageResult := []handlers.BlogPostPreviewDto{}
		err := json.Unmarshal([]byte(bodyFirstPage), &firstPageResult)
		assert.NoError(t, err)

		assert.Equal(t, 3, len(firstPageResult))
		assert.Equal(t, int64(5), firstPageResult[0].Id)
		assert.Equal(t, int64(4), firstPageResult[1].Id)
		assert.Equal(t, int64(3), firstPageResult[2].Id)
		assert.Equal(t, "generated_chat5", firstPageResult[0].Title)
		assert.Equal(t, "generated_chat4", firstPageResult[1].Title)
		assert.Equal(t, "generated_chat3", firstPageResult[2].Title)

		// get second page
		httpSecondPage, bodySecondPage, _ := request("GET", "/blog?startingFromItemId=3&size=3", nil, e)
		assert.Equal(t, http.StatusOK, httpSecondPage)
		assert.NotEmpty(t, bodySecondPage)

		secondPageResult := []handlers.BlogPostPreviewDto{}
		err = json.Unmarshal([]byte(bodySecondPage), &secondPageResult)
		assert.NoError(t, err)

		assert.Equal(t, 2, len(secondPageResult))
		assert.Equal(t, int64(2), secondPageResult[0].Id)
		assert.Equal(t, int64(1), secondPageResult[1].Id)
		assert.Equal(t, "generated_chat2", secondPageResult[0].Title)
		assert.Equal(t, "generated_chat1", secondPageResult[1].Title)
	})
}

func TestGetBlogsPaginatedSearch(t *testing.T) {
	runTest(t, func(e *echo.Echo) {
		// get first page
		httpFirstPage, bodyFirstPage, _ := request("GET", "/blog?startingFromItemId=6&size=3&searchString=gen", nil, e)
		assert.Equal(t, http.StatusOK, httpFirstPage)
		assert.NotEmpty(t, bodyFirstPage)

		firstPageResult := []handlers.BlogPostPreviewDto{}
		err := json.Unmarshal([]byte(bodyFirstPage), &firstPageResult)
		assert.NoError(t, err)

		assert.Equal(t, 3, len(firstPageResult))
		assert.Equal(t, int64(5), firstPageResult[0].Id)
		assert.Equal(t, int64(4), firstPageResult[1].Id)
		assert.Equal(t, int64(3), firstPageResult[2].Id)
		assert.Equal(t, "generated_chat5", firstPageResult[0].Title)
		assert.Equal(t, "generated_chat4", firstPageResult[1].Title)
		assert.Equal(t, "generated_chat3", firstPageResult[2].Title)

		// get second page
		httpSecondPage, bodySecondPage, _ := request("GET", "/blog?startingFromItemId=3&size=3", nil, e)
		assert.Equal(t, http.StatusOK, httpSecondPage)
		assert.NotEmpty(t, bodySecondPage)

		secondPageResult := []handlers.BlogPostPreviewDto{}
		err = json.Unmarshal([]byte(bodySecondPage), &secondPageResult)
		assert.NoError(t, err)

		assert.Equal(t, 2, len(secondPageResult))
		assert.Equal(t, int64(2), secondPageResult[0].Id)
		assert.Equal(t, int64(1), secondPageResult[1].Id)
		assert.Equal(t, "generated_chat2", secondPageResult[0].Title)
		assert.Equal(t, "generated_chat1", secondPageResult[1].Title)
	})
}

func TestGetBlogsHasHash(t *testing.T) {
	runTest(t, func(e *echo.Echo) {
		// get first page
		httpFirstPage, bodyFirstPage, _ := request("GET", "/blog?startingFromItemId=7&size=10&hasHash=true", nil, e)
		assert.Equal(t, http.StatusOK, httpFirstPage)
		assert.NotEmpty(t, bodyFirstPage)

		firstPageResult := []handlers.BlogPostPreviewDto{}
		err := json.Unmarshal([]byte(bodyFirstPage), &firstPageResult)
		assert.NoError(t, err)

		assert.Equal(t, 10, len(firstPageResult))
		assert.Equal(t, "generated_chat12", firstPageResult[0].Title)
		assert.Equal(t, "generated_chat11", firstPageResult[1].Title)
		assert.Equal(t, "generated_chat10", firstPageResult[2].Title)
		assert.Equal(t, "generated_chat9", firstPageResult[3].Title)
		assert.Equal(t, "generated_chat8", firstPageResult[4].Title)
		assert.Equal(t, "generated_chat7", firstPageResult[5].Title)
		assert.Equal(t, "generated_chat6", firstPageResult[6].Title)
		assert.Equal(t, "generated_chat5", firstPageResult[7].Title)
		assert.Equal(t, "generated_chat4", firstPageResult[8].Title)
		assert.Equal(t, "generated_chat3", firstPageResult[9].Title)
		assert.Equal(t, int64(12), firstPageResult[0].Id)
		assert.Equal(t, int64(11), firstPageResult[1].Id)
		assert.Equal(t, int64(10), firstPageResult[2].Id)
		assert.Equal(t, int64(9), firstPageResult[3].Id)
		assert.Equal(t, int64(8), firstPageResult[4].Id)
		assert.Equal(t, int64(7), firstPageResult[5].Id)
		assert.Equal(t, int64(6), firstPageResult[6].Id)
		assert.Equal(t, int64(5), firstPageResult[7].Id)
		assert.Equal(t, int64(4), firstPageResult[8].Id)
		assert.Equal(t, int64(3), firstPageResult[9].Id)
	})
}

func TestGetBlogsHasHashSearch(t *testing.T) {
	runTest(t, func(e *echo.Echo) {
		// get first page
		httpFirstPage, bodyFirstPage, _ := request("GET", "/blog?startingFromItemId=7&size=10&hasHash=true&searchString=gen", nil, e)
		assert.Equal(t, http.StatusOK, httpFirstPage)
		assert.NotEmpty(t, bodyFirstPage)

		firstPageResult := []handlers.BlogPostPreviewDto{}
		err := json.Unmarshal([]byte(bodyFirstPage), &firstPageResult)
		assert.NoError(t, err)

		assert.Equal(t, 10, len(firstPageResult))
		assert.Equal(t, "generated_chat12", firstPageResult[0].Title)
		assert.Equal(t, "generated_chat11", firstPageResult[1].Title)
		assert.Equal(t, "generated_chat10", firstPageResult[2].Title)
		assert.Equal(t, "generated_chat9", firstPageResult[3].Title)
		assert.Equal(t, "generated_chat8", firstPageResult[4].Title)
		assert.Equal(t, "generated_chat7", firstPageResult[5].Title)
		assert.Equal(t, "generated_chat6", firstPageResult[6].Title)
		assert.Equal(t, "generated_chat5", firstPageResult[7].Title)
		assert.Equal(t, "generated_chat4", firstPageResult[8].Title)
		assert.Equal(t, "generated_chat3", firstPageResult[9].Title)
		assert.Equal(t, int64(12), firstPageResult[0].Id)
		assert.Equal(t, int64(11), firstPageResult[1].Id)
		assert.Equal(t, int64(10), firstPageResult[2].Id)
		assert.Equal(t, int64(9), firstPageResult[3].Id)
		assert.Equal(t, int64(8), firstPageResult[4].Id)
		assert.Equal(t, int64(7), firstPageResult[5].Id)
		assert.Equal(t, int64(6), firstPageResult[6].Id)
		assert.Equal(t, int64(5), firstPageResult[7].Id)
		assert.Equal(t, int64(4), firstPageResult[8].Id)
		assert.Equal(t, int64(3), firstPageResult[9].Id)
	})
}
