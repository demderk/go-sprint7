package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type CafeTestSuite struct {
	suite.Suite
}

func (s *CafeTestSuite) TestCafeCountStrict() {
	sharedTests := []int{0, 1, 2}
	cities := []string{"moscow", "tula"}

	for _, city := range cities {
		for _, current := range sharedTests {
			s.Run(
				fmt.Sprintf("%s count: %v", city, current),
				func() {
					s.testCountCheck(city, current, current)
				},
			)
		}
	}
}

func (s *CafeTestSuite) TestCafeCountMax() {
	requests := []struct {
		city     string
		actual   int
		expected int
	}{
		{
			city:     "moscow",
			actual:   100,
			expected: len(cafeList["moscow"]),
		},
		{
			city:     "tula",
			actual:   100,
			expected: len(cafeList["tula"]),
		},
	}

	for _, current := range requests {
		s.Run(
			fmt.Sprintf("%s count: %v", current.city, current.actual),
			func() {
				s.testCountCheck(current.city, current.actual, current.expected)
			},
		)
	}
}

func (s *CafeTestSuite) TestCafeSearch() {
	requests := []struct {
		search    string
		wantCount int
	}{
		{
			search:    "фасоль",
			wantCount: 0,
		},
		{
			search:    "кофе",
			wantCount: 2,
		},
		{
			search:    "вилка",
			wantCount: 1,
		},
	}

	for _, request := range requests {
		s.Run(
			fmt.Sprintf("search match query %v count %v", request.search, request.wantCount),
			func() {
				s.testMatchSearchCount("moscow", request.search, request.wantCount)
			})
	}
}

func (s *CafeTestSuite) TestCafeWhenOk() {
	handler := http.HandlerFunc(mainHandle)

	requests := []string{
		"/cafe?count=2&city=moscow",
		"/cafe?city=tula",
		"/cafe?city=moscow&search=ложка",
	}
	for _, v := range requests {
		response := httptest.NewRecorder()
		req := httptest.NewRequest("GET", v, nil)

		handler.ServeHTTP(response, req)

		s.Assert().Equal(http.StatusOK, response.Code)
	}
}

func (s *CafeTestSuite) TestCafeNegative() {
	requests := []struct {
		endpoint     string
		errorMessage string
		errorCode    int
	}{
		{
			endpoint:     "/cafe",
			errorMessage: "unknown city",
			errorCode:    http.StatusBadRequest,
		},
		{
			endpoint:     "/cafe?city=omsk",
			errorMessage: "unknown city",
			errorCode:    http.StatusBadRequest,
		},
		{
			endpoint:     "/cafe?city=tula&count=na",
			errorMessage: "incorrect count",
			errorCode:    http.StatusBadRequest,
		},
	}

	for n, request := range requests {
		s.Run(
			fmt.Sprintf("negative request #%v", n),
			func() {
				handler := http.HandlerFunc(mainHandle)
				response := httptest.NewRecorder()
				req := httptest.NewRequest("GET", request.endpoint, nil)

				handler.ServeHTTP(response, req)

				s.Assert().Equal(request.errorCode, response.Code)
				s.Assert().Equal(request.errorMessage, strings.TrimSpace(response.Body.String()))
			},
		)
	}

}

func (s *CafeTestSuite) testMatchSearchCount(city string, query string, expected int) {
	handler := http.HandlerFunc(mainHandle)
	response := httptest.NewRecorder()

	endpoint := makeURL(
		"/cafe",
		map[string]string{
			"search": query,
			"city":   city,
		},
	)

	req := httptest.NewRequest("GET", endpoint, nil)

	handler.ServeHTTP(response, req)

	assert.Equal(s.T(), http.StatusOK, response.Code)
	responseBytes, err := io.ReadAll(response.Body)
	if err != nil {
		s.T().Error(fmt.Errorf("can't read response: %w", err))
	}

	resultString := string(responseBytes)

	resultString = strings.ToLower(resultString)
	queryString := strings.ToLower(query)

	if strings.TrimSpace(resultString) == "" {
		assert.Equal(s.T(), expected, 0)
		return
	}

	resultArray := strings.Split(resultString, ",")
	assert.Equal(s.T(), expected, len(resultArray))
	assert.True(s.T(), strings.Contains(resultString, queryString))
}

func (s *CafeTestSuite) testCountCheck(city string, actual int, expected int) {
	handler := http.HandlerFunc(mainHandle)

	endpoint := makeURL(
		"/cafe",
		map[string]string{
			"count": strconv.Itoa(actual),
			"city":  city,
		},
	)

	response := httptest.NewRecorder()
	req := httptest.NewRequest("GET", endpoint, nil)

	handler.ServeHTTP(response, req)

	assert.Equal(s.T(), http.StatusOK, response.Code)
	responseBytes, err := io.ReadAll(response.Body)
	if err != nil {
		s.T().Error(fmt.Errorf("can't read response: %w", err))
	}

	resultString := string(responseBytes)

	if strings.TrimSpace(resultString) == "" {
		s.Assert().Equal(expected, 0)
		return
	}

	resultArray := strings.Split(resultString, ",")
	s.Assert().Equal(expected, len(resultArray))

}

func TestCafeTestSuite(t *testing.T) {
	suite.Run(t, new(CafeTestSuite))
}

func makeURL(endpoint string, args map[string]string) string {
	url, err := url.Parse(endpoint)
	if err != nil {
		panic(fmt.Errorf("url was failed to build: %w", err))
	}
	query := url.Query()
	for k, v := range args {
		query.Add(k, v)
	}
	url.RawQuery = query.Encode()
	return url.String()
}
