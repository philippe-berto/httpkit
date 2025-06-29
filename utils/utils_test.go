package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testStruct struct {
	ID   int
	Name string
}

var (
	server   chi.Router
	testCase = testStruct{
		ID:   1,
		Name: "example",
	}
)

func TestMain(m *testing.M) {
	server = chi.NewRouter()
	server.Use(middleware.StripSlashes)
	server.Get("/fault", func(w http.ResponseWriter, r *http.Request) {
		_ = Fault(w, http.StatusBadRequest, InvalidParam, "fault test")
	})

	server.Get("/fault_extra", func(w http.ResponseWriter, r *http.Request) {
		_ = FaultWithData(w, http.StatusBadRequest, InvalidParam, "fault test", map[string]interface{}{"test": true})
	})

	server.Get("/success_body", func(w http.ResponseWriter, r *http.Request) {
		_ = WriteBody(w, http.StatusOK, testCase)
	})

	server.Get("/error_body", func(w http.ResponseWriter, r *http.Request) {
		_ = WriteBody(w, http.StatusOK, math.Inf(1))
	})

	server.Get("/int/{accountID}", func(w http.ResponseWriter, r *http.Request) {
		id, err := ParseIDParam(w, r, "accountID")
		if err == nil {
			_, _ = w.Write([]byte(strconv.FormatInt(id, 10)))
		}
	})

	server.Get("/float/{accountID}", func(w http.ResponseWriter, r *http.Request) {
		id, err := ParseFloatParam(w, r, "accountID")
		if err == nil {
			_, _ = w.Write([]byte(fmt.Sprintf("%g", id)))
		}
	})

	server.Get("/uuid/{accountID}", func(w http.ResponseWriter, r *http.Request) {
		id, err := ParseParamUUID(w, r, "accountID")
		if err == nil {
			_, _ = w.Write([]byte(id.String()))
		}
	})

	server.Get("/string/{accountID}", func(w http.ResponseWriter, r *http.Request) {
		param := ParseParam(w, r, "accountID")
		_, _ = w.Write([]byte(param))
	})

	server.Get("/float-query", func(w http.ResponseWriter, r *http.Request) {
		id, err := ParseFloatQuery(w, r, "accountID")
		if err == nil {
			_, _ = w.Write([]byte(fmt.Sprintf("%g", id)))
		}
	})

	code := m.Run()
	os.Exit(code)
}

func TestUtil_Fault(t *testing.T) {
	t.Run("should return a valid error ", func(t *testing.T) {
		responseWriter := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/fault", nil)
		require.NoError(t, err)

		server.ServeHTTP(responseWriter, request)

		reqErr := Error{}
		err = json.Unmarshal(responseWriter.Body.Bytes(), &reqErr)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, responseWriter.Code)
		assert.Equal(t, InvalidParam, reqErr.Code)
		assert.Equal(t, "fault test", reqErr.Message)
	})
}

func TestUtil_FaultWithData(t *testing.T) {
	t.Run("should return a valid erro with data ", func(t *testing.T) {
		responseWriter := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/fault_extra", nil)
		require.NoError(t, err)

		server.ServeHTTP(responseWriter, request)

		reqErr := Error{}
		err = json.Unmarshal(responseWriter.Body.Bytes(), &reqErr)
		require.NoError(t, err)

		data, err := ConvertToMap(responseWriter.Body.String())
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, responseWriter.Code)
		assert.Contains(t, data, ErrorCode)
		assert.Contains(t, data, ErrorMessage)
		assert.Contains(t, data, "test")

		assert.Equal(t, InvalidParam, data[ErrorCode])
		assert.Equal(t, "fault test", data[ErrorMessage])
		assert.Equal(t, true, data["test"])
	})
}

func TestUtil_WriteJSON(t *testing.T) {
	testCase := testStruct{
		ID:   1,
		Name: "example",
	}
	testCaseJSON, _ := json.Marshal(testCase)

	t.Run("should return valid json", func(t *testing.T) {
		w := httptest.NewRecorder()

		WriteJSON(w, 200, testCaseJSON)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, testCaseJSON, w.Body.Bytes())
		assert.Equal(t, ApplicationJSON, w.Header().Get(ContentType))
	})

	t.Run("should return valid json", func(t *testing.T) {
		w := httptest.NewRecorder()

		WriteJSON(w, 200, testCaseJSON)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, testCaseJSON, w.Body.Bytes())
		assert.Equal(t, ApplicationJSON, w.Header().Get(ContentType))
	})
}

func TestUtil_ReadBody(t *testing.T) {
	testCaseJSON, _ := json.Marshal(testCase)

	t.Run("should convert valid struct ", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(testCaseJSON))

		var testValue testStruct
		err := ReadBody(r, &testValue)
		assert.NoError(t, err)
		assert.Equal(t, testCase, testValue)
	})

	t.Run("should convert valid struct ", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader([]byte("asdasd")))

		var testValue testStruct
		err := ReadBody(r, &testValue)
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidBody, err)
	})
}

func TestUtil_WriteBody(t *testing.T) {
	testCaseJSON, _ := json.Marshal(testCase)
	t.Run("should successfully ", func(t *testing.T) {
		responseWriter := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/success_body", nil)
		server.ServeHTTP(responseWriter, request)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, responseWriter.Code)
		assert.Equal(t, testCaseJSON, responseWriter.Body.Bytes())
	})

	t.Run("should return error ", func(t *testing.T) {
		responseWriter := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/error_body", nil)
		require.NoError(t, err)

		server.ServeHTTP(responseWriter, request)

		reqErr := Error{}
		err = json.Unmarshal(responseWriter.Body.Bytes(), &reqErr)
		require.NoError(t, err)

		assert.Equal(t, http.StatusInternalServerError, responseWriter.Code)
		assert.Equal(t, FaultCodeInternalServerError, reqErr.Code)
		assert.Equal(t, "json: unsupported value: +Inf", reqErr.Message)
	})
}

func TestUtil_ParseIDParam(t *testing.T) {
	t.Run("should successfully return id ", func(t *testing.T) {
		responseWriter := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/int/1111", nil)
		server.ServeHTTP(responseWriter, request)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, responseWriter.Code)
		assert.Equal(t, "1111", responseWriter.Body.String())
	})

	t.Run("should invalid format ", func(t *testing.T) {
		responseWriter := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/int/asdad", nil)
		require.NoError(t, err)

		server.ServeHTTP(responseWriter, request)

		reqErr := Error{}
		err = json.Unmarshal(responseWriter.Body.Bytes(), &reqErr)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, responseWriter.Code)
		assert.Equal(t, InvalidParam, reqErr.Code)
		assert.Equal(t, "accountID is invalid type", reqErr.Message)
	})
}

func TestUtil_ParseFloatParam(t *testing.T) {
	t.Run("should successfully return id ", func(t *testing.T) {
		responseWriter := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/float/12.222", nil)
		server.ServeHTTP(responseWriter, request)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, responseWriter.Code)
		assert.Equal(t, "12.222", responseWriter.Body.String())
	})

	t.Run("should invalid format ", func(t *testing.T) {
		responseWriter := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/float/asdad", nil)
		require.NoError(t, err)

		server.ServeHTTP(responseWriter, request)

		reqErr := Error{}
		err = json.Unmarshal(responseWriter.Body.Bytes(), &reqErr)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, responseWriter.Code)
		assert.Equal(t, InvalidParam, reqErr.Code)
		assert.Equal(t, "accountID is invalid type", reqErr.Message)
	})
}

func TestUtil_ParseParamUUID(t *testing.T) {
	fmt.Println(uuid.New().String())
	t.Run("should successfully return uuid", func(t *testing.T) {
		responseWriter := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/uuid/3b2ec853-8f82-4c98-b0ab-b3183815c8aa", nil)
		server.ServeHTTP(responseWriter, request)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, responseWriter.Code)
		assert.Equal(t, "3b2ec853-8f82-4c98-b0ab-b3183815c8aa", responseWriter.Body.String())
	})

	t.Run("should invalid format with int ", func(t *testing.T) {
		responseWriter := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/uuid/1111", nil)
		require.NoError(t, err)

		server.ServeHTTP(responseWriter, request)

		reqErr := Error{}
		err = json.Unmarshal(responseWriter.Body.Bytes(), &reqErr)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, responseWriter.Code)
		assert.Equal(t, InvalidParam, reqErr.Code)
		assert.Equal(t, "invalid UUID length: 4", reqErr.Message)
	})

	t.Run("should invalid format string", func(t *testing.T) {
		responseWriter := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/uuid/asdad", nil)
		require.NoError(t, err)

		server.ServeHTTP(responseWriter, request)

		reqErr := Error{}
		err = json.Unmarshal(responseWriter.Body.Bytes(), &reqErr)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, responseWriter.Code)
		assert.Equal(t, InvalidParam, reqErr.Code)
		assert.Equal(t, "invalid UUID length: 5", reqErr.Message)
	})
}

func TestUtil_ParseParam(t *testing.T) {
	fmt.Println(uuid.New().String())
	t.Run("should successfully return ", func(t *testing.T) {
		responseWriter := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/string/test", nil)
		server.ServeHTTP(responseWriter, request)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, responseWriter.Code)
		assert.Equal(t, "test", responseWriter.Body.String())
	})
}

func TestUtil_ParseFloatQuery(t *testing.T) {
	t.Run("should successfully return id ", func(t *testing.T) {
		responseWriter := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/float-query?accountID=12.222", nil)
		server.ServeHTTP(responseWriter, request)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, responseWriter.Code)
		assert.Equal(t, "12.222", responseWriter.Body.String())
	})

	t.Run("should not find", func(t *testing.T) {
		responseWriter := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/float-query", nil)
		require.NoError(t, err)

		server.ServeHTTP(responseWriter, request)

		reqErr := Error{}
		err = json.Unmarshal(responseWriter.Body.Bytes(), &reqErr)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, responseWriter.Code)
		assert.Equal(t, InvalidParam, reqErr.Code)
		assert.Equal(t, "accountID was not not found", reqErr.Message)
	})

	t.Run("should invalid format ", func(t *testing.T) {
		responseWriter := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/float-query?accountID=asdad", nil)
		require.NoError(t, err)

		server.ServeHTTP(responseWriter, request)

		reqErr := Error{}
		err = json.Unmarshal(responseWriter.Body.Bytes(), &reqErr)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, responseWriter.Code)
		assert.Equal(t, InvalidParam, reqErr.Code)
		assert.Equal(t, "accountID is invalid type", reqErr.Message)
	})
}
