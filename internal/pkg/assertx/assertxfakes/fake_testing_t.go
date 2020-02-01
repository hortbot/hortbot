// Code generated by counterfeiter. DO NOT EDIT.
package assertxfakes

import (
	"sync"

	"github.com/hortbot/hortbot/internal/pkg/assertx"
)

type FakeTestingT struct {
	FailStub        func()
	failMutex       sync.RWMutex
	failArgsForCall []struct {
	}
	FailNowStub        func()
	failNowMutex       sync.RWMutex
	failNowArgsForCall []struct {
	}
	HelperStub        func()
	helperMutex       sync.RWMutex
	helperArgsForCall []struct {
	}
	LogStub        func(...interface{})
	logMutex       sync.RWMutex
	logArgsForCall []struct {
		arg1 []interface{}
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeTestingT) Fail() {
	fake.failMutex.Lock()
	fake.failArgsForCall = append(fake.failArgsForCall, struct {
	}{})
	fake.recordInvocation("Fail", []interface{}{})
	fake.failMutex.Unlock()
	if fake.FailStub != nil {
		fake.FailStub()
	}
}

func (fake *FakeTestingT) FailCallCount() int {
	fake.failMutex.RLock()
	defer fake.failMutex.RUnlock()
	return len(fake.failArgsForCall)
}

func (fake *FakeTestingT) FailCalls(stub func()) {
	fake.failMutex.Lock()
	defer fake.failMutex.Unlock()
	fake.FailStub = stub
}

func (fake *FakeTestingT) FailNow() {
	fake.failNowMutex.Lock()
	fake.failNowArgsForCall = append(fake.failNowArgsForCall, struct {
	}{})
	fake.recordInvocation("FailNow", []interface{}{})
	fake.failNowMutex.Unlock()
	if fake.FailNowStub != nil {
		fake.FailNowStub()
	}
}

func (fake *FakeTestingT) FailNowCallCount() int {
	fake.failNowMutex.RLock()
	defer fake.failNowMutex.RUnlock()
	return len(fake.failNowArgsForCall)
}

func (fake *FakeTestingT) FailNowCalls(stub func()) {
	fake.failNowMutex.Lock()
	defer fake.failNowMutex.Unlock()
	fake.FailNowStub = stub
}

func (fake *FakeTestingT) Helper() {
	fake.helperMutex.Lock()
	fake.helperArgsForCall = append(fake.helperArgsForCall, struct {
	}{})
	fake.recordInvocation("Helper", []interface{}{})
	fake.helperMutex.Unlock()
	if fake.HelperStub != nil {
		fake.HelperStub()
	}
}

func (fake *FakeTestingT) HelperCallCount() int {
	fake.helperMutex.RLock()
	defer fake.helperMutex.RUnlock()
	return len(fake.helperArgsForCall)
}

func (fake *FakeTestingT) HelperCalls(stub func()) {
	fake.helperMutex.Lock()
	defer fake.helperMutex.Unlock()
	fake.HelperStub = stub
}

func (fake *FakeTestingT) Log(arg1 ...interface{}) {
	fake.logMutex.Lock()
	fake.logArgsForCall = append(fake.logArgsForCall, struct {
		arg1 []interface{}
	}{arg1})
	fake.recordInvocation("Log", []interface{}{arg1})
	fake.logMutex.Unlock()
	if fake.LogStub != nil {
		fake.LogStub(arg1...)
	}
}

func (fake *FakeTestingT) LogCallCount() int {
	fake.logMutex.RLock()
	defer fake.logMutex.RUnlock()
	return len(fake.logArgsForCall)
}

func (fake *FakeTestingT) LogCalls(stub func(...interface{})) {
	fake.logMutex.Lock()
	defer fake.logMutex.Unlock()
	fake.LogStub = stub
}

func (fake *FakeTestingT) LogArgsForCall(i int) []interface{} {
	fake.logMutex.RLock()
	defer fake.logMutex.RUnlock()
	argsForCall := fake.logArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeTestingT) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.failMutex.RLock()
	defer fake.failMutex.RUnlock()
	fake.failNowMutex.RLock()
	defer fake.failNowMutex.RUnlock()
	fake.helperMutex.RLock()
	defer fake.helperMutex.RUnlock()
	fake.logMutex.RLock()
	defer fake.logMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeTestingT) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ assertx.TestingT = new(FakeTestingT)
