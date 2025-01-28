package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/login", login)
	return r
}

func TestLoginEndpoint(t *testing.T) {
	router := setupRouter()

	tests := []struct {
		name           string
		requestBody    LoginRequest
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "Valid Beta User Login",
			requestBody: LoginRequest{
				Username: "betauser",
				Password: "betauser",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response LoginResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)

				// Verify user details
				assert.Equal(t, "betauser", response.User.Username)
				assert.Equal(t, "acme global", response.User.Company)
				assert.True(t, response.User.BetaAccess)
				assert.Empty(t, response.User.Password) // Password should not be in response

				// Verify JWT token
				token, err := jwt.Parse(response.Token, func(token *jwt.Token) (interface{}, error) {
					return jwtSecret, nil
				})
				assert.NoError(t, err)
				assert.True(t, token.Valid)

				claims, ok := token.Claims.(jwt.MapClaims)
				assert.True(t, ok)
				assert.Equal(t, "betauser", claims["username"])
				assert.Equal(t, "acme global", claims["company"])
				assert.Equal(t, true, claims["beta_access"])
			},
		},
		{
			name: "Valid Normal User Login",
			requestBody: LoginRequest{
				Username: "normaluser",
				Password: "normaluser",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response LoginResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)

				assert.Equal(t, "normaluser", response.User.Username)
				assert.Equal(t, "generic co", response.User.Company)
				assert.False(t, response.User.BetaAccess)
				assert.Empty(t, response.User.Password)
			},
		},
		{
			name: "Invalid Credentials",
			requestBody: LoginRequest{
				Username: "wronguser",
				Password: "wrongpass",
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, "Invalid credentials", response["error"])
			},
		},
		{
			name: "Invalid Request - Missing Password",
			requestBody: LoginRequest{
				Username: "betauser",
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, "Invalid request", response["error"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.requestBody)
			assert.NoError(t, err)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			tt.checkResponse(t, w)
		})
	}
}

func TestFindUser(t *testing.T) {
	tests := []struct {
		name          string
		username      string
		password      string
		expectedUser  *User
		shouldBeFound bool
	}{
		{
			name:          "Valid Beta User",
			username:      "betauser",
			password:      "betauser",
			shouldBeFound: true,
			expectedUser: &User{
				Username:   "betauser",
				Password:   "betauser",
				Company:    "acme global",
				BetaAccess: true,
			},
		},
		{
			name:          "Valid Normal User",
			username:      "normaluser",
			password:      "normaluser",
			shouldBeFound: true,
			expectedUser: &User{
				Username:   "normaluser",
				Password:   "normaluser",
				Company:    "generic co",
				BetaAccess: false,
			},
		},
		{
			name:          "Invalid Username",
			username:      "nonexistent",
			password:      "betauser",
			shouldBeFound: false,
		},
		{
			name:          "Invalid Password",
			username:      "betauser",
			password:      "wrongpass",
			shouldBeFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := findUser(tt.username, tt.password)
			if tt.shouldBeFound {
				assert.NotNil(t, user)
				assert.Equal(t, tt.expectedUser.Username, user.Username)
				assert.Equal(t, tt.expectedUser.Password, user.Password)
				assert.Equal(t, tt.expectedUser.Company, user.Company)
				assert.Equal(t, tt.expectedUser.BetaAccess, user.BetaAccess)
			} else {
				assert.Nil(t, user)
			}
		})
	}
}
