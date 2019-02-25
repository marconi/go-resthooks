package resthooks

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
)

var routePat = regexp.MustCompile(`(subscribe|unsubscribe)/?(\d+)?/?`)

type handle func(http.ResponseWriter, *http.Request)

type route struct {
	method  string
	handler handle
}

type Handler struct {
	rh     *Resthook
	routes map[string]*route
}

func NewHandler(rh *Resthook) http.Handler {
	handler := &Handler{rh: rh}
	handler.routes = map[string]*route{
		"subscribe": &route{
			method:  "POST",
			handler: handler.Subscribe,
		},
		"unsubscribe": &route{
			method:  "DELETE",
			handler: handler.Unsubscribe,
		},
	}
	return handler
}

func (h *Handler) Subscribe(w http.ResponseWriter, r *http.Request) {
	// NOTE: Here we are decoding to Subscription but if we have
	// more data aside from event and target_url, we can instead
	// decode to map field like Subsciption.extras so we
	// can preserve those as well.
	s := new(Subscription)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(s); err != nil {
		http.Error(w, "Invalid subscribe data.", http.StatusBadRequest)
		return
	}

	if err := h.rh.Save(s); err != nil {
		http.Error(w, "Error creating subscription.", http.StatusInternalServerError)
		return
	}

	out, err := json.Marshal(s)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err = w.Write(out); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	result := routePat.FindStringSubmatch(r.URL.Path)
	id, err := strconv.ParseInt(result[2], 10, 32)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.rh.DeleteById(int(id)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	result := routePat.FindStringSubmatch(r.URL.Path)
	if result == nil {
		http.NotFound(w, r)
		return
	}

	handler, ok := h.routes[result[1]]
	if !ok {
		http.NotFound(w, r)
		return
	}

	if handler.method != r.Method {
		http.NotFound(w, r)
		return
	}

	handler.handler(w, r)
}

// TODO: implement, list, get and update subscription endpoints
