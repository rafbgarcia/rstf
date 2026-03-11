package rstf

import (
	"encoding/json"
	"sort"
	"strings"
	"sync"
)

type SubscriptionKey string

type LiveEventType string

const (
	LiveEventQueryResult LiveEventType = "query_result"
	LiveEventQueryError  LiveEventType = "query_error"
)

type LiveError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type LiveEvent struct {
	Type           LiveEventType `json:"type"`
	SubscriptionID string        `json:"subscriptionId"`
	Data           any           `json:"data,omitempty"`
	Error          *LiveError    `json:"error,omitempty"`
}

type LiveSubscription struct {
	ClientID       string
	SubscriptionID string
	Key            SubscriptionKey
	Execute        func() (any, error)
}

type liveClient struct {
	ch chan LiveEvent
}

// LiveHub stores in-memory live-query subscriptions for a single app instance.
type LiveHub struct {
	mu           sync.RWMutex
	clients      map[string]*liveClient
	subsByClient map[string]map[string]*LiveSubscription
	subsByKey    map[SubscriptionKey]map[string]*LiveSubscription
}

// NewLiveHub creates an empty in-memory live-query hub.
func NewLiveHub() *LiveHub {
	return &LiveHub{
		clients:      map[string]*liveClient{},
		subsByClient: map[string]map[string]*LiveSubscription{},
		subsByKey:    map[SubscriptionKey]map[string]*LiveSubscription{},
	}
}

// NewSubscriptionKey builds a canonical key for a query function and params.
func NewSubscriptionKey(routeName, queryName string, params map[string]string) SubscriptionKey {
	if len(params) == 0 {
		return SubscriptionKey("query:" + routeName + ":" + queryName + ":{}")
	}

	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	buf := strings.Builder{}
	buf.WriteString("{")
	for i, key := range keys {
		if i > 0 {
			buf.WriteString(",")
		}
		keyJSON, _ := json.Marshal(key)
		valJSON, _ := json.Marshal(params[key])
		buf.Write(keyJSON)
		buf.WriteString(":")
		buf.Write(valJSON)
	}
	buf.WriteString("}")
	return SubscriptionKey("query:" + routeName + ":" + queryName + ":" + buf.String())
}

// Connect registers a live SSE connection for a client ID.
func (h *LiveHub) Connect(clientID string) (<-chan LiveEvent, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if existing, ok := h.clients[clientID]; ok {
		close(existing.ch)
	}

	client := &liveClient{
		ch: make(chan LiveEvent, 32),
	}
	h.clients[clientID] = client

	return client.ch, func() {
		h.mu.Lock()
		defer h.mu.Unlock()

		current, ok := h.clients[clientID]
		if !ok || current != client {
			return
		}
		delete(h.clients, clientID)
		close(client.ch)
	}
}

// Register stores or replaces a live subscription for the client.
func (h *LiveHub) Register(sub *LiveSubscription) {
	if h == nil || sub == nil || sub.ClientID == "" || sub.SubscriptionID == "" || sub.Execute == nil {
		return
	}

	compositeID := sub.ClientID + ":" + sub.SubscriptionID

	h.mu.Lock()
	defer h.mu.Unlock()

	if clientSubs, ok := h.subsByClient[sub.ClientID]; ok {
		if existing, ok := clientSubs[sub.SubscriptionID]; ok {
			if keySubs, ok := h.subsByKey[existing.Key]; ok {
				delete(keySubs, compositeID)
				if len(keySubs) == 0 {
					delete(h.subsByKey, existing.Key)
				}
			}
		}
	}

	clientSubs := h.subsByClient[sub.ClientID]
	if clientSubs == nil {
		clientSubs = map[string]*LiveSubscription{}
		h.subsByClient[sub.ClientID] = clientSubs
	}
	clientSubs[sub.SubscriptionID] = sub

	keySubs := h.subsByKey[sub.Key]
	if keySubs == nil {
		keySubs = map[string]*LiveSubscription{}
		h.subsByKey[sub.Key] = keySubs
	}
	keySubs[compositeID] = sub
}

// Unregister removes a single live subscription.
func (h *LiveHub) Unregister(clientID, subscriptionID string) {
	if h == nil || clientID == "" || subscriptionID == "" {
		return
	}

	compositeID := clientID + ":" + subscriptionID

	h.mu.Lock()
	defer h.mu.Unlock()

	clientSubs, ok := h.subsByClient[clientID]
	if !ok {
		return
	}
	sub, ok := clientSubs[subscriptionID]
	if !ok {
		return
	}
	delete(clientSubs, subscriptionID)
	if len(clientSubs) == 0 {
		delete(h.subsByClient, clientID)
	}

	if keySubs, ok := h.subsByKey[sub.Key]; ok {
		delete(keySubs, compositeID)
		if len(keySubs) == 0 {
			delete(h.subsByKey, sub.Key)
		}
	}
}

// Replay reruns all subscriptions for a client and emits their latest snapshots.
func (h *LiveHub) Replay(clientID string) {
	if h == nil || clientID == "" {
		return
	}
	h.mu.RLock()
	clientSubs := h.subsByClient[clientID]
	subs := make([]*LiveSubscription, 0, len(clientSubs))
	for _, sub := range clientSubs {
		subs = append(subs, sub)
	}
	h.mu.RUnlock()

	for _, sub := range subs {
		h.refresh(sub)
	}
}

// Invalidate reruns all subscriptions for the given keys.
func (h *LiveHub) Invalidate(keys ...SubscriptionKey) {
	if h == nil || len(keys) == 0 {
		return
	}

	h.mu.RLock()
	snapshot := map[string]*LiveSubscription{}
	for _, key := range keys {
		for compositeID, sub := range h.subsByKey[key] {
			snapshot[compositeID] = sub
		}
	}
	h.mu.RUnlock()

	for _, sub := range snapshot {
		go h.refresh(sub)
	}
}

func (h *LiveHub) refresh(sub *LiveSubscription) {
	if sub == nil || sub.Execute == nil {
		return
	}

	data, err := sub.Execute()
	if err != nil {
		re := requestErrorFrom(err)
		h.publish(sub.ClientID, LiveEvent{
			Type:           LiveEventQueryError,
			SubscriptionID: sub.SubscriptionID,
			Error: &LiveError{
				Code:    string(re.Code),
				Message: re.Message,
			},
		})
		return
	}

	h.publish(sub.ClientID, LiveEvent{
		Type:           LiveEventQueryResult,
		SubscriptionID: sub.SubscriptionID,
		Data:           data,
	})
}

func (h *LiveHub) publish(clientID string, event LiveEvent) {
	h.mu.RLock()
	client := h.clients[clientID]
	h.mu.RUnlock()
	if client == nil {
		return
	}

	select {
	case client.ch <- event:
	default:
	}
}
