package resthooks

import (
	"bytes"
	"io"
)

const (
	STATUS_PENDING Status = iota
	STATUS_SUCCESS
	STATUS_FAILED
)

type Status int

// ResthookStore defines APIs for subscription's
// CRUD operations. Using this you can customise
// where and how data is actually being stored.
// Or that you actually use soft-delete
// to preserve history of subscription.
type ResthookStore interface {
	// Creates subscription if it doesn't have id
	// and populates the id, otherwise updates it.
	Save(*Subscription) error

	FindById(int) (*Subscription, error)
	FindByUserId(int, string) (*Subscription, error)
	DeleteById(int) error
}

type Subscription struct {
	Id        int    `json:"id"`
	UserId    int    `json:"user_id"`
	Event     string `json:"event"`
	TargetUrl string `json:"target_url"`
}

type Notification struct {
	Subscription *Subscription
	Data         []byte
	Status       Status
	Retries      int
}

// Returns encapsulated data as io.Reader
func (n Notification) asReader() io.Reader {
	return bytes.NewReader(n.Data)
}
