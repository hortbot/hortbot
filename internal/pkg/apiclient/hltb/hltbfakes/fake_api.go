// Code generated by counterfeiter. DO NOT EDIT.
package hltbfakes

import (
	"context"
	"sync"

	"github.com/hortbot/hortbot/internal/pkg/apiclient/hltb"
)

type FakeAPI struct {
	SearchGameStub        func(context.Context, string) (*hltb.Game, error)
	searchGameMutex       sync.RWMutex
	searchGameArgsForCall []struct {
		arg1 context.Context
		arg2 string
	}
	searchGameReturns struct {
		result1 *hltb.Game
		result2 error
	}
	searchGameReturnsOnCall map[int]struct {
		result1 *hltb.Game
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeAPI) SearchGame(arg1 context.Context, arg2 string) (*hltb.Game, error) {
	fake.searchGameMutex.Lock()
	ret, specificReturn := fake.searchGameReturnsOnCall[len(fake.searchGameArgsForCall)]
	fake.searchGameArgsForCall = append(fake.searchGameArgsForCall, struct {
		arg1 context.Context
		arg2 string
	}{arg1, arg2})
	fake.recordInvocation("SearchGame", []interface{}{arg1, arg2})
	fake.searchGameMutex.Unlock()
	if fake.SearchGameStub != nil {
		return fake.SearchGameStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.searchGameReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeAPI) SearchGameCallCount() int {
	fake.searchGameMutex.RLock()
	defer fake.searchGameMutex.RUnlock()
	return len(fake.searchGameArgsForCall)
}

func (fake *FakeAPI) SearchGameCalls(stub func(context.Context, string) (*hltb.Game, error)) {
	fake.searchGameMutex.Lock()
	defer fake.searchGameMutex.Unlock()
	fake.SearchGameStub = stub
}

func (fake *FakeAPI) SearchGameArgsForCall(i int) (context.Context, string) {
	fake.searchGameMutex.RLock()
	defer fake.searchGameMutex.RUnlock()
	argsForCall := fake.searchGameArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeAPI) SearchGameReturns(result1 *hltb.Game, result2 error) {
	fake.searchGameMutex.Lock()
	defer fake.searchGameMutex.Unlock()
	fake.SearchGameStub = nil
	fake.searchGameReturns = struct {
		result1 *hltb.Game
		result2 error
	}{result1, result2}
}

func (fake *FakeAPI) SearchGameReturnsOnCall(i int, result1 *hltb.Game, result2 error) {
	fake.searchGameMutex.Lock()
	defer fake.searchGameMutex.Unlock()
	fake.SearchGameStub = nil
	if fake.searchGameReturnsOnCall == nil {
		fake.searchGameReturnsOnCall = make(map[int]struct {
			result1 *hltb.Game
			result2 error
		})
	}
	fake.searchGameReturnsOnCall[i] = struct {
		result1 *hltb.Game
		result2 error
	}{result1, result2}
}

func (fake *FakeAPI) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.searchGameMutex.RLock()
	defer fake.searchGameMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeAPI) recordInvocation(key string, args []interface{}) {
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

var _ hltb.API = new(FakeAPI)
