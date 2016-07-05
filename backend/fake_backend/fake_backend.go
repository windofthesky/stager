// This file was generated by counterfeiter
package fake_backend

import (
	"sync"

	"code.cloudfoundry.org/bbs/models"
	"code.cloudfoundry.org/runtimeschema/cc_messages"
	"code.cloudfoundry.org/stager/backend"
)

type FakeBackend struct {
	BuildRecipeStub        func(stagingGuid string, request cc_messages.StagingRequestFromCC) (*models.TaskDefinition, string, string, error)
	buildRecipeMutex       sync.RWMutex
	buildRecipeArgsForCall []struct {
		stagingGuid string
		request     cc_messages.StagingRequestFromCC
	}
	buildRecipeReturns struct {
		result1 *models.TaskDefinition
		result2 string
		result3 string
		result4 error
	}
	BuildStagingResponseStub        func(*models.TaskCallbackResponse) (cc_messages.StagingResponseForCC, error)
	buildStagingResponseMutex       sync.RWMutex
	buildStagingResponseArgsForCall []struct {
		arg1 *models.TaskCallbackResponse
	}
	buildStagingResponseReturns struct {
		result1 cc_messages.StagingResponseForCC
		result2 error
	}
}

func (fake *FakeBackend) BuildRecipe(stagingGuid string, request cc_messages.StagingRequestFromCC) (*models.TaskDefinition, string, string, error) {
	fake.buildRecipeMutex.Lock()
	fake.buildRecipeArgsForCall = append(fake.buildRecipeArgsForCall, struct {
		stagingGuid string
		request     cc_messages.StagingRequestFromCC
	}{stagingGuid, request})
	fake.buildRecipeMutex.Unlock()
	if fake.BuildRecipeStub != nil {
		return fake.BuildRecipeStub(stagingGuid, request)
	} else {
		return fake.buildRecipeReturns.result1, fake.buildRecipeReturns.result2, fake.buildRecipeReturns.result3, fake.buildRecipeReturns.result4
	}
}

func (fake *FakeBackend) BuildRecipeCallCount() int {
	fake.buildRecipeMutex.RLock()
	defer fake.buildRecipeMutex.RUnlock()
	return len(fake.buildRecipeArgsForCall)
}

func (fake *FakeBackend) BuildRecipeArgsForCall(i int) (string, cc_messages.StagingRequestFromCC) {
	fake.buildRecipeMutex.RLock()
	defer fake.buildRecipeMutex.RUnlock()
	return fake.buildRecipeArgsForCall[i].stagingGuid, fake.buildRecipeArgsForCall[i].request
}

func (fake *FakeBackend) BuildRecipeReturns(result1 *models.TaskDefinition, result2 string, result3 string, result4 error) {
	fake.BuildRecipeStub = nil
	fake.buildRecipeReturns = struct {
		result1 *models.TaskDefinition
		result2 string
		result3 string
		result4 error
	}{result1, result2, result3, result4}
}

func (fake *FakeBackend) BuildStagingResponse(arg1 *models.TaskCallbackResponse) (cc_messages.StagingResponseForCC, error) {
	fake.buildStagingResponseMutex.Lock()
	fake.buildStagingResponseArgsForCall = append(fake.buildStagingResponseArgsForCall, struct {
		arg1 *models.TaskCallbackResponse
	}{arg1})
	fake.buildStagingResponseMutex.Unlock()
	if fake.BuildStagingResponseStub != nil {
		return fake.BuildStagingResponseStub(arg1)
	} else {
		return fake.buildStagingResponseReturns.result1, fake.buildStagingResponseReturns.result2
	}
}

func (fake *FakeBackend) BuildStagingResponseCallCount() int {
	fake.buildStagingResponseMutex.RLock()
	defer fake.buildStagingResponseMutex.RUnlock()
	return len(fake.buildStagingResponseArgsForCall)
}

func (fake *FakeBackend) BuildStagingResponseArgsForCall(i int) *models.TaskCallbackResponse {
	fake.buildStagingResponseMutex.RLock()
	defer fake.buildStagingResponseMutex.RUnlock()
	return fake.buildStagingResponseArgsForCall[i].arg1
}

func (fake *FakeBackend) BuildStagingResponseReturns(result1 cc_messages.StagingResponseForCC, result2 error) {
	fake.BuildStagingResponseStub = nil
	fake.buildStagingResponseReturns = struct {
		result1 cc_messages.StagingResponseForCC
		result2 error
	}{result1, result2}
}

var _ backend.Backend = new(FakeBackend)
