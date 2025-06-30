package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const (
	InvalidBody        = "invalid_body"
	InvalidParam       = "invalid_param"
	InvalidCredentials = "invalid_credentials"
	InternalCode       = "internal_server_error"

	ContentType                  = "Content-Type"
	ApplicationJSON              = "application/json; charset=utf-8"
	FaultCodeInternalServerError = "internal_server_error"

	ErrorCode    = "code"
	ErrorMsg     = "msg"
	ErrorMessage = "message"
)

var (
	ErrEmptyBody   = errors.New("body is empty")
	ErrInvalidBody = errors.New("body is invalid")
)

type Error struct {
	Code    string `json:"code"`
	Msg     string `json:"msg"`
	Message string `json:"message"`
}

func ReadBody(r *http.Request, v interface{}) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return ErrInvalidBody
	}

	if len(body) == 0 {
		return ErrEmptyBody
	}

	if err := json.Unmarshal(body, v); err != nil {
		return ErrInvalidBody
	}

	return nil
}

func WriteBody(w http.ResponseWriter, statusCode int, body interface{}) error {
	result, err := json.Marshal(body)
	if err != nil {
		_ = Fault(w, http.StatusInternalServerError, FaultCodeInternalServerError, err.Error())

		return err
	}

	WriteJSON(w, statusCode, result)

	return nil
}

func WriteJSON(w http.ResponseWriter, statusCode int, body []byte) {
	w.Header().Set(ContentType, ApplicationJSON)
	w.WriteHeader(statusCode)
	_, _ = w.Write(body)
}

func ConvertToMap(jsonStr string) (map[string]interface{}, error) {
	var result map[string]interface{}

	err := json.Unmarshal([]byte(jsonStr), &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func Fault(w http.ResponseWriter, httpStatus int, code, message string) error {
	w.Header().Set(ContentType, ApplicationJSON)
	w.WriteHeader(httpStatus)

	response := make(map[string]interface{})
	response[ErrorCode] = code
	response[ErrorMsg] = message
	response[ErrorMessage] = message

	enc := json.NewEncoder(w)

	return enc.Encode(response)
}

func FaultWithData(w http.ResponseWriter, httpStatus int, code, message string, additionalData map[string]interface{}) error {
	w.Header().Set(ContentType, ApplicationJSON)
	w.WriteHeader(httpStatus)

	response := make(map[string]interface{})
	response[ErrorCode] = code
	response[ErrorMsg] = message
	response[ErrorMessage] = message

	for key, value := range additionalData {
		response[key] = value
	}

	enc := json.NewEncoder(w)

	return enc.Encode(response)
}

func ParseParam(w http.ResponseWriter, r *http.Request, param string) string {
	return chi.URLParam(r, param)
}

func ParseIDParam(w http.ResponseWriter, r *http.Request, param string) (int64, error) {
	parsedParam := chi.URLParam(r, param)
	if parsedParam == "" {
		_ = Fault(w, http.StatusBadRequest, InvalidParam, fmt.Sprintf("%s was not not found", param))

		return 0, errors.New("parameter nof found")
	}

	parsedID, err := strconv.Atoi(parsedParam)
	if err != nil {
		_ = Fault(w, http.StatusBadRequest, InvalidParam, fmt.Sprintf("%s is invalid type", param))

		return 0, errors.New("parameter invalid")
	}

	return int64(parsedID), nil
}

func ParseFloatParam(w http.ResponseWriter, r *http.Request, param string) (float64, error) {
	parsedParam := chi.URLParam(r, param)
	if parsedParam == "" {
		_ = Fault(w, http.StatusBadRequest, InvalidParam, fmt.Sprintf("%s was not not found", param))

		return 0, errors.New("parameter nof found")
	}

	parsedID, err := strconv.ParseFloat(parsedParam, 64)
	if err != nil {
		_ = Fault(w, http.StatusBadRequest, InvalidParam, fmt.Sprintf("%s is invalid type", param))

		return 0, errors.New("parameter invalid")
	}

	return parsedID, nil
}

func ParseParamUUID(w http.ResponseWriter, r *http.Request, param string) (uuid.UUID, error) {
	parsedParam, err := uuid.Parse(chi.URLParam(r, param))
	if err != nil {
		_ = Fault(w, http.StatusBadRequest, InvalidParam, err.Error())
	}

	return parsedParam, err
}

func ParseFloatQuery(w http.ResponseWriter, r *http.Request, param string) (float64, error) {
	parsedParam := r.URL.Query().Get(param)
	if parsedParam == "" {
		_ = Fault(w, http.StatusBadRequest, InvalidParam, fmt.Sprintf("%s was not not found", param))

		return 0, errors.New("parameter nof found")
	}

	parsedID, err := strconv.ParseFloat(parsedParam, 64)
	if err != nil {
		_ = Fault(w, http.StatusBadRequest, InvalidParam, fmt.Sprintf("%s is invalid type", param))

		return 0, errors.New("parameter invalid")
	}

	return parsedID, nil
}
