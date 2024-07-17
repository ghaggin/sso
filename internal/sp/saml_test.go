package sp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_requireAuth(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	newSessionManager()

	req, err := http.NewRequest("GET", "/", nil)
	require.Nil(err)

	responseRecorder := httptest.NewRecorder()

	calledNext := false
	testHandler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		calledNext = true
	})

	handler := sessionManager.LoadAndSave(requireAuth(testHandler))

	handler.ServeHTTP(responseRecorder, req)
	assert.False(calledNext)
	assert.Equal(http.StatusSeeOther, responseRecorder.Code)
	assert.Equal("/saml/login", responseRecorder.Result().Header.Get("Location"))

}

func Test_requireAuth2(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	newSessionManager()

	r, err := http.NewRequest("GET", "/", nil)
	require.Nil(err)
	rr := httptest.NewRecorder()

	putUser := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sessionManager.Put(r.Context(), "user", &User{UID: "test"})
			next.ServeHTTP(w, r)
		})
	}

	calledNext := false
	nextHandler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		calledNext = true
	})

	handler := sessionManager.LoadAndSave(putUser(requireAuth(nextHandler)))

	handler.ServeHTTP(rr, r)
	assert.True(calledNext)
	assert.Equal(http.StatusOK, rr.Code)
}
