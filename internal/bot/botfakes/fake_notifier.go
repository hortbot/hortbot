// Code generated by counterfeiter. DO NOT EDIT.
package botfakes

import (
	"sync"

	"github.com/hortbot/hortbot/internal/bot"
)

type FakeNotifier struct {
	NotifyChannelUpdatesStub        func(string) error
	notifyChannelUpdatesMutex       sync.RWMutex
	notifyChannelUpdatesArgsForCall []struct {
		arg1 string
	}
	notifyChannelUpdatesReturns struct {
		result1 error
	}
	notifyChannelUpdatesReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeNotifier) NotifyChannelUpdates(arg1 string) error {
	fake.notifyChannelUpdatesMutex.Lock()
	ret, specificReturn := fake.notifyChannelUpdatesReturnsOnCall[len(fake.notifyChannelUpdatesArgsForCall)]
	fake.notifyChannelUpdatesArgsForCall = append(fake.notifyChannelUpdatesArgsForCall, struct {
		arg1 string
	}{arg1})
	fake.recordInvocation("NotifyChannelUpdates", []interface{}{arg1})
	fake.notifyChannelUpdatesMutex.Unlock()
	if fake.NotifyChannelUpdatesStub != nil {
		return fake.NotifyChannelUpdatesStub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.notifyChannelUpdatesReturns
	return fakeReturns.result1
}

func (fake *FakeNotifier) NotifyChannelUpdatesCallCount() int {
	fake.notifyChannelUpdatesMutex.RLock()
	defer fake.notifyChannelUpdatesMutex.RUnlock()
	return len(fake.notifyChannelUpdatesArgsForCall)
}

func (fake *FakeNotifier) NotifyChannelUpdatesCalls(stub func(string) error) {
	fake.notifyChannelUpdatesMutex.Lock()
	defer fake.notifyChannelUpdatesMutex.Unlock()
	fake.NotifyChannelUpdatesStub = stub
}

func (fake *FakeNotifier) NotifyChannelUpdatesArgsForCall(i int) string {
	fake.notifyChannelUpdatesMutex.RLock()
	defer fake.notifyChannelUpdatesMutex.RUnlock()
	argsForCall := fake.notifyChannelUpdatesArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeNotifier) NotifyChannelUpdatesReturns(result1 error) {
	fake.notifyChannelUpdatesMutex.Lock()
	defer fake.notifyChannelUpdatesMutex.Unlock()
	fake.NotifyChannelUpdatesStub = nil
	fake.notifyChannelUpdatesReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeNotifier) NotifyChannelUpdatesReturnsOnCall(i int, result1 error) {
	fake.notifyChannelUpdatesMutex.Lock()
	defer fake.notifyChannelUpdatesMutex.Unlock()
	fake.NotifyChannelUpdatesStub = nil
	if fake.notifyChannelUpdatesReturnsOnCall == nil {
		fake.notifyChannelUpdatesReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.notifyChannelUpdatesReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeNotifier) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.notifyChannelUpdatesMutex.RLock()
	defer fake.notifyChannelUpdatesMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeNotifier) recordInvocation(key string, args []interface{}) {
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

var _ bot.Notifier = new(FakeNotifier)
