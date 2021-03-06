package apitest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-pg/pg/v9/orm"

	"vobook/cmd/server/routes"
	"vobook/config"
	"vobook/database"
	"vobook/database/factories"
	"vobook/database/models"
	authtoken "vobook/domain/auth_token"
	"vobook/logger"
	"vobook/services/mail"
	"vobook/tests/assert"
	"vobook/utils"
)

var (
	DB        orm.DB
	AuthUser  *models.User
	authToken string
	router    *gin.Engine
	Conf      *config.Config
)

type Headers map[string]string

type Request struct {
	method       string
	Path         string
	Body         interface{}
	bodyReader   io.Reader
	Headers      Headers
	BindResponse interface{}
	AssertStatus int
	IsPublic     bool
}

const ContentTypeJSON = "application/json"

func init() {
	// changing working dir to vobook/bin
	// just want to use main .config.yaml, thats why
	_ = os.Chdir("../../../bin")
	Conf = config.Get()

	// override config
	Conf.LogsFilePath = "tests.log"
	Conf.Mail.Driver = mail.TestDriverName
	Conf.App.Env = config.TestEnv

	logger.Setup()

	db := database.Conn()
	_, _ = db.Exec("SET AUTOCOMMIT TO OFF")
	tx, err := db.Conn().Begin()
	if err != nil {
		panic(err)
	}
	database.SetDB(tx)
	DB = database.ORM()

	mail.InitDrivers()

	router = gin.New()
	routes.Register(router)
}

// makeRequest makes request (Duh...)
func makeRequest(t *testing.T, r Request) *httptest.ResponseRecorder {
	req, err := http.NewRequest(
		r.method,
		r.Path,
		r.bodyReader,
	)
	assert.NotError(t, err)
	t.Log(fmt.Sprintf("[%d] %s %s", r.AssertStatus, r.method, r.Path))
	req.Header.Set("Content-Type", ContentTypeJSON)
	req.Header.Set("Accept", ContentTypeJSON)
	req.Header.Set("X-Client", "1")

	for k, v := range r.Headers {
		req.Header.Set(k, v)
	}

	if !r.IsPublic {
		if authToken == "" {
			Login(t)
		}

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	return w
}

// TestRequest sends given request,
// asserts that status is correct
// binds response and returns response
func TestRequest(t *testing.T, req Request) *httptest.ResponseRecorder {
	var body []byte
	var err error

	if p := config.Get().ApiBasePath; p != "" {
		req.Path = path.Join(p, req.Path)
	}

	if !strings.HasSuffix(req.Path, "/") && !strings.Contains(req.Path, "/?") {
		req.Path += "/"
	}

	if len(req.Headers) == 0 {
		req.Headers = Headers{}
	}

	if req.Body != nil {
		body, err = json.Marshal(req.Body)
		assert.NotError(t, err)
	}

	req.bodyReader = strings.NewReader(string(body))
	resp := makeRequest(t, req)
	if resp.Code != req.AssertStatus {
		t.Errorf(`%s "%s" responds with %d, expecting %d`, req.method, req.Path, resp.Code, req.AssertStatus)
		t.Errorf("Response:\n%s", resp.Body.String())
		if req.Body != nil {
			t.Errorf("Request:\n%+v", req.bodyReader)
		}
		t.FailNow()
	}

	if responseContentType(resp) == ContentTypeJSON {
		err = json.Unmarshal(resp.Body.Bytes(), &req.BindResponse)
		if err != nil {
			t.Log(resp.Body.String())
			t.Fatal(err)
		}
	}

	return resp
}

func Login(t *testing.T) *models.User {
	if AuthUser != nil && authToken != "" {
		return AuthUser
	}
	user, err := factories.CreateUser()
	assert.NotError(t, err)
	LoginAs(t, &user)

	return &user
}

func responseContentType(resp *httptest.ResponseRecorder) string {
	ct, ok := resp.Header()["Content-Type"]
	if !ok || len(ct) == 0 {
		return ""
	}

	frags := strings.Split(ct[0], ";")
	return frags[0]
}

func User(t *testing.T) *models.User {
	if AuthUser == nil {
		Login(t)
	}

	return AuthUser
}

func LoginAs(t *testing.T, user *models.User) {
	token := &models.AuthToken{
		UserID:   user.ID,
		ClientID: models.VueClient,
	}
	err := authtoken.Create(token)
	assert.NotError(t, err)

	AuthUser = user
	authToken = authtoken.Sign(token)
}

func ReLogin(t *testing.T) {
	Logout()
	Login(t)
}

func Logout() {
	AuthUser = nil
	authToken = ""
}

// POST is a wrapper for TestRequest
func POST(t *testing.T, req Request) *httptest.ResponseRecorder {
	req.method = "POST"
	return TestRequest(t, req)
}

// PUT is a wrapper for TestRequest
func PUT(t *testing.T, req Request) *httptest.ResponseRecorder {
	req.method = "PUT"
	return TestRequest(t, req)
}

// PATCH is a wrapper for TestRequest
func PATCH(t *testing.T, req Request) *httptest.ResponseRecorder {
	req.method = "PATCH"
	return TestRequest(t, req)
}

// DELETE is a wrapper for TestRequest
func DELETE(t *testing.T, req Request) *httptest.ResponseRecorder {
	req.method = "DELETE"
	return TestRequest(t, req)
}

// GET is a wrapper for TestRequest
func GET(t *testing.T, req Request) *httptest.ResponseRecorder {
	req.method = "GET"
	return TestRequest(t, req)
}

// TestCreate makes "create" POST request that expects response type of response arg and 201 status code
func TestCreate(t *testing.T, path string, body, response interface{}) *httptest.ResponseRecorder {
	req := Request{
		Path:         path,
		Body:         body,
		BindResponse: response,
		AssertStatus: http.StatusCreated,
	}

	return POST(t, req)
}

// TestUpdate makes "update" request
func TestUpdate(t *testing.T, path string, body, response interface{}) *httptest.ResponseRecorder {
	req := Request{
		Path:         path,
		Body:         body,
		BindResponse: response,
		AssertStatus: http.StatusOK,
	}

	return PUT(t, req)
}

// Patch makes "patch" request
func Patch(t *testing.T, path string, body, response interface{}) *httptest.ResponseRecorder {
	req := Request{
		Path:         path,
		Body:         body,
		BindResponse: response,
		AssertStatus: http.StatusOK,
	}

	return PATCH(t, req)
}

// TestDelete makes "delete" request
func TestDelete(t *testing.T, path string, response interface{}) *httptest.ResponseRecorder {
	req := Request{
		Path:         path,
		BindResponse: response,
		AssertStatus: http.StatusOK,
	}

	return DELETE(t, req)
}

// TestGet makes simple "get" request
func TestGet(t *testing.T, path string, response interface{}) *httptest.ResponseRecorder {
	req := Request{
		Path:         path,
		BindResponse: response,
		AssertStatus: http.StatusOK,
	}

	return GET(t, req)
}

// TestSearch makes "get" request with query params
func TestSearch(t *testing.T, path string, request, response interface{}) *httptest.ResponseRecorder {
	query, err := utils.StructToQueryString(request)
	if err != nil {
		t.Fatal(err.Error())
	}

	path = fmt.Sprintf(`%s/?%s`, path, query)

	req := Request{
		Path:         path,
		BindResponse: response,
		AssertStatus: http.StatusOK,
	}

	return GET(t, req)
}
