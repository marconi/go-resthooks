package resthooks_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/marconi/resthooks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type FakeStore struct {
	mock.Mock
}

func (fs *FakeStore) Save(s *resthooks.Subscription) error {
	args := fs.Called(s)
	return args.Error(0)
}

func (fs *FakeStore) FindById(id int) (*resthooks.Subscription, error) {
	args := fs.Called(id)
	return args.Get(0).(*resthooks.Subscription), args.Error(1)
}

func (fs *FakeStore) DeleteById(id int) error {
	args := fs.Called(id)
	return args.Error(0)
}

func (fs *FakeStore) FindByUserId(id int, event string) (*resthooks.Subscription, error) {
	args := fs.Called(id, event)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*resthooks.Subscription), args.Error(1)
}

func TestSubscribe(t *testing.T) {
	data := map[string]interface{}{
		"user_id":    1,
		"event":      "post_created",
		"target_url": "http://somewhere.com/hooks/notify",
	}
	in, err := json.Marshal(data)
	assert.NotEmpty(t, in)
	assert.Empty(t, err)

	req, err := http.NewRequest("POST", "/hooks/subscribe", bytes.NewBuffer(in))
	assert.NotEmpty(t, req)
	assert.Empty(t, err)

	store := new(FakeStore)
	rr := httptest.NewRecorder()
	rh := resthooks.NewResthook(store)
	defer rh.Close()

	// ensure save is called with non-saved
	// subscription and that it returns
	// subscription with an id and no error
	unsaved := &resthooks.Subscription{
		UserId:    data["user_id"].(int),
		Event:     data["event"].(string),
		TargetUrl: data["target_url"].(string),
	}
	store.On("Save", unsaved).Run(func(args mock.Arguments) {
		s := args.Get(0).(*resthooks.Subscription)
		s.Id = 1
	}).Return(nil)

	rh.Handler().ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, rr.Header()["Content-Type"][0], "application/json")

	// ensure was saved
	store.AssertNumberOfCalls(t, "Save", 1)
	store.AssertExpectations(t)

	// ensure it returns newly saved subscription
	saved := new(resthooks.Subscription)
	decoder := json.NewDecoder(rr.Body)
	err = decoder.Decode(saved)
	assert.Empty(t, err)
	assert.Equal(t, saved.Id, 1)
	assert.Equal(t, saved.UserId, unsaved.UserId)
	assert.Equal(t, saved.Event, unsaved.Event)
	assert.Equal(t, saved.TargetUrl, unsaved.TargetUrl)
}

func TestSubscribeWithInvalidMethod(t *testing.T) {
	data := map[string]string{
		"event":      "post_created",
		"target_url": "http://somewhere.com/hooks/notify",
	}
	in, err := json.Marshal(data)
	assert.NotEmpty(t, in)
	assert.Empty(t, err)

	req, err := http.NewRequest("PUT", "/hooks/subscribe", bytes.NewBuffer(in))
	assert.NotEmpty(t, req)
	assert.Empty(t, err)

	rr := httptest.NewRecorder()
	rh := resthooks.NewResthook(new(FakeStore))
	defer rh.Close()

	rh.Handler().ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestUnsubscribe(t *testing.T) {
	req, err := http.NewRequest("DELETE", "/hooks/unsubscribe/1", nil)
	assert.NotEmpty(t, req)
	assert.Empty(t, err)

	s := &resthooks.Subscription{
		Id:        1,
		UserId:    1,
		Event:     "post_created",
		TargetUrl: "http://somewhere.com/hooks/notify",
	}
	store := new(FakeStore)
	store.On("FindById", 1).Return(s, nil)
	store.On("DeleteById", 1).Return(nil)

	rr := httptest.NewRecorder()
	rh := resthooks.NewResthook(store)
	defer rh.Close()

	rh.Handler().ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// ensure was saved
	store.AssertNumberOfCalls(t, "FindById", 1)
	store.AssertNumberOfCalls(t, "DeleteById", 1)
	store.AssertExpectations(t)
}

func TestUnsubscribeWithInvalidMethod(t *testing.T) {
	req, err := http.NewRequest("GET", "/hooks/unsubscribe/1", nil)
	assert.NotEmpty(t, req)
	assert.Empty(t, err)

	rr := httptest.NewRecorder()
	rh := resthooks.NewResthook(new(FakeStore))
	defer rh.Close()

	rh.Handler().ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}
