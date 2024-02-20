// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package assertxmocks

import (
	"sync"

	"github.com/hortbot/hortbot/internal/pkg/assertx"
)

// Ensure, that TestingTMock does implement assertx.TestingT.
// If this is not the case, regenerate this file with moq.
var _ assertx.TestingT = &TestingTMock{}

// TestingTMock is a mock implementation of assertx.TestingT.
//
//	func TestSomethingThatUsesTestingT(t *testing.T) {
//
//		// make and configure a mocked assertx.TestingT
//		mockedTestingT := &TestingTMock{
//			FailFunc: func()  {
//				panic("mock out the Fail method")
//			},
//			FailNowFunc: func()  {
//				panic("mock out the FailNow method")
//			},
//			HelperFunc: func()  {
//				panic("mock out the Helper method")
//			},
//			LogFunc: func(args ...interface{})  {
//				panic("mock out the Log method")
//			},
//		}
//
//		// use mockedTestingT in code that requires assertx.TestingT
//		// and then make assertions.
//
//	}
type TestingTMock struct {
	// FailFunc mocks the Fail method.
	FailFunc func()

	// FailNowFunc mocks the FailNow method.
	FailNowFunc func()

	// HelperFunc mocks the Helper method.
	HelperFunc func()

	// LogFunc mocks the Log method.
	LogFunc func(args ...interface{})

	// calls tracks calls to the methods.
	calls struct {
		// Fail holds details about calls to the Fail method.
		Fail []struct {
		}
		// FailNow holds details about calls to the FailNow method.
		FailNow []struct {
		}
		// Helper holds details about calls to the Helper method.
		Helper []struct {
		}
		// Log holds details about calls to the Log method.
		Log []struct {
			// Args is the args argument value.
			Args []interface{}
		}
	}
	lockFail    sync.RWMutex
	lockFailNow sync.RWMutex
	lockHelper  sync.RWMutex
	lockLog     sync.RWMutex
}

// Fail calls FailFunc.
func (mock *TestingTMock) Fail() {
	if mock.FailFunc == nil {
		panic("TestingTMock.FailFunc: method is nil but TestingT.Fail was just called")
	}
	callInfo := struct {
	}{}
	mock.lockFail.Lock()
	mock.calls.Fail = append(mock.calls.Fail, callInfo)
	mock.lockFail.Unlock()
	mock.FailFunc()
}

// FailCalls gets all the calls that were made to Fail.
// Check the length with:
//
//	len(mockedTestingT.FailCalls())
func (mock *TestingTMock) FailCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockFail.RLock()
	calls = mock.calls.Fail
	mock.lockFail.RUnlock()
	return calls
}

// FailNow calls FailNowFunc.
func (mock *TestingTMock) FailNow() {
	if mock.FailNowFunc == nil {
		panic("TestingTMock.FailNowFunc: method is nil but TestingT.FailNow was just called")
	}
	callInfo := struct {
	}{}
	mock.lockFailNow.Lock()
	mock.calls.FailNow = append(mock.calls.FailNow, callInfo)
	mock.lockFailNow.Unlock()
	mock.FailNowFunc()
}

// FailNowCalls gets all the calls that were made to FailNow.
// Check the length with:
//
//	len(mockedTestingT.FailNowCalls())
func (mock *TestingTMock) FailNowCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockFailNow.RLock()
	calls = mock.calls.FailNow
	mock.lockFailNow.RUnlock()
	return calls
}

// Helper calls HelperFunc.
func (mock *TestingTMock) Helper() {
	if mock.HelperFunc == nil {
		panic("TestingTMock.HelperFunc: method is nil but TestingT.Helper was just called")
	}
	callInfo := struct {
	}{}
	mock.lockHelper.Lock()
	mock.calls.Helper = append(mock.calls.Helper, callInfo)
	mock.lockHelper.Unlock()
	mock.HelperFunc()
}

// HelperCalls gets all the calls that were made to Helper.
// Check the length with:
//
//	len(mockedTestingT.HelperCalls())
func (mock *TestingTMock) HelperCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockHelper.RLock()
	calls = mock.calls.Helper
	mock.lockHelper.RUnlock()
	return calls
}

// Log calls LogFunc.
func (mock *TestingTMock) Log(args ...interface{}) {
	if mock.LogFunc == nil {
		panic("TestingTMock.LogFunc: method is nil but TestingT.Log was just called")
	}
	callInfo := struct {
		Args []interface{}
	}{
		Args: args,
	}
	mock.lockLog.Lock()
	mock.calls.Log = append(mock.calls.Log, callInfo)
	mock.lockLog.Unlock()
	mock.LogFunc(args...)
}

// LogCalls gets all the calls that were made to Log.
// Check the length with:
//
//	len(mockedTestingT.LogCalls())
func (mock *TestingTMock) LogCalls() []struct {
	Args []interface{}
} {
	var calls []struct {
		Args []interface{}
	}
	mock.lockLog.RLock()
	calls = mock.calls.Log
	mock.lockLog.RUnlock()
	return calls
}