package resthooks_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/marconi/resthooks"
)

type SampleData struct {
	Id int
}

func TestNotifyInvalidUser(t *testing.T) {
	s := &resthooks.Subscription{
		Id:        1,
		UserId:    1,
		Event:     "post_created",
		TargetUrl: "",
	}

	store := new(FakeStore)
	store.On("FindByUserId", s.UserId, s.Event).Return(nil, errors.New("User not fould."))

	rh := resthooks.NewResthook(store)
	err := rh.Notify(1, "post_created", s)
	assert.NotEmpty(t, err, "Should return error on user not found")
	store.AssertExpectations(t)
}

func TestNotifyPostSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	s := &resthooks.Subscription{
		Id:        1,
		UserId:    1,
		Event:     "post_created",
		TargetUrl: ts.URL,
	}

	store := new(FakeStore)
	store.On("FindByUserId", s.UserId, s.Event).Return(s, nil)

	rh := resthooks.NewResthook(store)
	defer rh.Close()

	// ensure results are piped to reader channel
	go func() {
		for data := range rh.GetResults() {
			assert.Equal(t, data.Status, resthooks.STATUS_SUCCESS)
			assert.Equal(t, data.Subscription.Id, s.Id)
		}
	}()

	err := rh.Notify(1, "post_created", new(SampleData))
	assert.Empty(t, err, "Should return no error on successful post")
	store.AssertExpectations(t)
}

func TestNotifyPostGone(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGone)
	}))
	defer ts.Close()

	s := &resthooks.Subscription{
		Id:        1,
		UserId:    1,
		Event:     "post_created",
		TargetUrl: ts.URL,
	}

	store := new(FakeStore)
	store.On("FindByUserId", s.UserId, s.Event).Return(s, nil)
	store.On("FindById", s.Id).Return(s, nil)
	store.On("DeleteById", s.Id).Return(nil)

	rh := resthooks.NewResthook(store)
	defer rh.Close()

	err := rh.Notify(1, "post_created", new(SampleData))
	assert.Empty(t, err, "Should return no error on post with gone response")
	store.AssertExpectations(t)
	store.AssertNumberOfCalls(t, "FindByUserId", 1)
	store.AssertNumberOfCalls(t, "FindById", 1)
	store.AssertNumberOfCalls(t, "DeleteById", 1)
}

func TestNotifyPostRetryMax(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer ts.Close()

	s := &resthooks.Subscription{
		Id:        1,
		UserId:    1,
		Event:     "post_created",
		TargetUrl: ts.URL,
	}

	store := new(FakeStore)
	store.On("FindByUserId", s.UserId, s.Event).Return(s, nil)

	config := resthooks.Config{
		InitialRetry:    1,
		RetryMultiplier: 1,
		MaxRetry:        2,
	}
	rh := resthooks.NewResthook(store, config)
	defer rh.Close()

	// ensure results are piped to reader channel
	continueChan := make(chan bool)
	go func() {
		for data := range rh.GetResults() {
			assert.Equal(t, data.Status, resthooks.STATUS_FAILED)
			assert.Equal(t, data.Subscription.Id, s.Id)
			assert.Equal(t, data.Retries, 2)

			// once we receive notification, continue with the test
			continueChan <- true
		}
	}()

	err := rh.Notify(1, "post_created", new(SampleData))

	// wait till we reach max attempts
	<-continueChan

	assert.Empty(t, err, "Should return no error on post with retry")
	store.AssertExpectations(t)
}

func TestNotifyPostRetry(t *testing.T) {
	// first request gets 400 so it'll be retried
	// first retry also gets 400
	// and second retry gets 200
	var currentStatus int
	status := []int{http.StatusBadRequest, http.StatusBadRequest, http.StatusOK}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		currentStatus, status = status[0], status[1:]
		w.WriteHeader(currentStatus)
	}))
	defer ts.Close()

	s := &resthooks.Subscription{
		Id:        1,
		UserId:    1,
		Event:     "post_created",
		TargetUrl: ts.URL,
	}

	store := new(FakeStore)
	store.On("FindByUserId", s.UserId, s.Event).Return(s, nil)

	config := resthooks.Config{
		InitialRetry:    1,
		RetryMultiplier: 1,
		MaxRetry:        2,
	}
	rh := resthooks.NewResthook(store, config)
	defer rh.Close()

	// ensure results are piped to reader channel
	continueChan := make(chan bool)
	go func() {
		for data := range rh.GetResults() {
			assert.Equal(t, resthooks.STATUS_SUCCESS, data.Status)
			assert.Equal(t, s.Id, data.Subscription.Id)
			assert.Equal(t, 2, data.Retries)

			// once we receive notification, continue with the test
			continueChan <- true
		}
	}()

	err := rh.Notify(1, "post_created", new(SampleData))

	// wait till we reach max attempts
	<-continueChan

	assert.Empty(t, err, "Should return no error on post with retry")
	store.AssertExpectations(t)
}
