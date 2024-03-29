// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package tinyurlmocks

import (
	"context"
	"sync"

	"github.com/hortbot/hortbot/internal/pkg/apiclient/tinyurl"
)

// Ensure, that APIMock does implement tinyurl.API.
// If this is not the case, regenerate this file with moq.
var _ tinyurl.API = &APIMock{}

// APIMock is a mock implementation of tinyurl.API.
//
//	func TestSomethingThatUsesAPI(t *testing.T) {
//
//		// make and configure a mocked tinyurl.API
//		mockedAPI := &APIMock{
//			ShortenFunc: func(ctx context.Context, url string) (string, error) {
//				panic("mock out the Shorten method")
//			},
//		}
//
//		// use mockedAPI in code that requires tinyurl.API
//		// and then make assertions.
//
//	}
type APIMock struct {
	// ShortenFunc mocks the Shorten method.
	ShortenFunc func(ctx context.Context, url string) (string, error)

	// calls tracks calls to the methods.
	calls struct {
		// Shorten holds details about calls to the Shorten method.
		Shorten []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// URL is the url argument value.
			URL string
		}
	}
	lockShorten sync.RWMutex
}

// Shorten calls ShortenFunc.
func (mock *APIMock) Shorten(ctx context.Context, url string) (string, error) {
	if mock.ShortenFunc == nil {
		panic("APIMock.ShortenFunc: method is nil but API.Shorten was just called")
	}
	callInfo := struct {
		Ctx context.Context
		URL string
	}{
		Ctx: ctx,
		URL: url,
	}
	mock.lockShorten.Lock()
	mock.calls.Shorten = append(mock.calls.Shorten, callInfo)
	mock.lockShorten.Unlock()
	return mock.ShortenFunc(ctx, url)
}

// ShortenCalls gets all the calls that were made to Shorten.
// Check the length with:
//
//	len(mockedAPI.ShortenCalls())
func (mock *APIMock) ShortenCalls() []struct {
	Ctx context.Context
	URL string
} {
	var calls []struct {
		Ctx context.Context
		URL string
	}
	mock.lockShorten.RLock()
	calls = mock.calls.Shorten
	mock.lockShorten.RUnlock()
	return calls
}
