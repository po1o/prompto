package auth

import (
	"errors"
	"testing"

	"github.com/po1o/prompto/src/cache"
	runtime_ "github.com/po1o/prompto/src/runtime/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYtdma_Authenticate(t *testing.T) {
	t.Cleanup(func() {
		cache.DeleteAll(cache.Device)
		cache.DeleteAll(cache.Session)
	})

	testCases := []struct {
		name                 string
		requestCodeResponse  string
		requestCodeError     error
		requestTokenResponse string
		requestTokenError    error
		expectedError        error
		expectedToken        string
		shouldSetToken       bool
	}{
		{
			name:                 "successful authentication",
			requestCodeResponse:  `{"code":"test-code-123"}`,
			requestCodeError:     nil,
			requestTokenResponse: `{"token":"test-token-456"}`,
			requestTokenError:    nil,
			expectedError:        nil,
			expectedToken:        "test-token-456",
			shouldSetToken:       true,
		},
		{
			name:                 "request code fails",
			requestCodeResponse:  "",
			requestCodeError:     errors.New("failed to request code"),
			requestTokenResponse: "",
			requestTokenError:    nil,
			expectedError:        errors.New("failed to request code"),
			expectedToken:        "",
			shouldSetToken:       false,
		},
		{
			name:                 "request token fails",
			requestCodeResponse:  `{"code":"test-code-123"}`,
			requestCodeError:     nil,
			requestTokenResponse: "",
			requestTokenError:    errors.New("failed to request token"),
			expectedError:        errors.New("failed to request token"),
			expectedToken:        "",
			shouldSetToken:       false,
		},
		{
			name:                 "invalid code response JSON",
			requestCodeResponse:  `{"invalid":"json"}`,
			requestCodeError:     nil,
			requestTokenResponse: "",
			requestTokenError:    nil,
			expectedError:        errors.New("unexpected end of JSON input"),
			expectedToken:        "",
			shouldSetToken:       false,
		},
		{
			name:                 "invalid token response JSON",
			requestCodeResponse:  `{"code":"test-code-123"}`,
			requestCodeError:     nil,
			requestTokenResponse: `{"invalid":"json"}`,
			requestTokenError:    nil,
			expectedError:        errors.New("received empty token"),
			expectedToken:        "",
			shouldSetToken:       false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cache.DeleteAll(cache.Device)
			cache.DeleteAll(cache.Session)

			env := &runtime_.Environment{}

			env.On("HTTPRequest", codeURL).Return([]byte(tc.requestCodeResponse), tc.requestCodeError)
			env.On("HTTPRequest", tokenURL).Return([]byte(tc.requestTokenResponse), tc.requestTokenError)

			ytmda := &Ytmda{
				model: model{
					env: env,
				},
			}

			ytmda.Authenticate()

			if tc.expectedError != nil {
				require.NotNil(t, ytmda.err)
				assert.Equal(t, tc.expectedError.Error(), ytmda.err.Error())
			} else {
				assert.Nil(t, ytmda.err)
			}

			token, ok := cache.Get[string](cache.Device, YTMDATOKEN)
			assert.Equal(t, tc.shouldSetToken, ok)
			assert.Equal(t, tc.expectedToken, token)

			_, ok = cache.Get[string](cache.Session, YTMDATOKEN)
			assert.False(t, ok)
		})
	}
}
