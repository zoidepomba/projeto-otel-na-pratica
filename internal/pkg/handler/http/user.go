// Copyright Dose de Telemetria GmbH
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"encoding/json"
	"net/http"

	"github.com/dosedetelemetria/projeto-otel-na-pratica/internal/pkg/log"
	"github.com/dosedetelemetria/projeto-otel-na-pratica/internal/pkg/model"
	"github.com/dosedetelemetria/projeto-otel-na-pratica/internal/pkg/store"
)

// UserHandler is an HTTP handler that performs CRUD operations for model.User using a store.User
type UserHandler struct {
	store store.User
}

// NewUserHandler returns a new UserHandler
func NewUserHandler(store store.User) *UserHandler {
	return &UserHandler{
		store: store,
	}
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	logger := log.NewLogeer()
	logger.Info("listing users")
	users, err := h.store.List(r.Context())
	if err != nil {
		logger.Error("error listing users")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(users)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logger.Info("users listed")
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	logger := log.NewLogeer()
	logger.Info("creating user")
	user := &model.User{}
	if err := json.NewDecoder(r.Body).Decode(user); err != nil {
		logger.Error("invalid request payload")
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	
	created, err := h.store.Create(r.Context(), user)
	logger.Info("created user: %s com o id %s", created.Name, created.ID)
	if err != nil {
		logger.Error("error creating user")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(created)
	if err != nil {
		logger.Error("error encoding user")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
	logger := log.NewLogeer()

	logger.Info("getting user")

	id := r.PathValue("id")
	user, err := h.store.Get(r.Context(), id)

	if err != nil {
		logger.Error("error getting user")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if user == nil {
		logger.Error("userID not found %s", id)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	logger.Info("user found Name: %s ID: %s", user.Name, user.ID)
	err = json.NewEncoder(w).Encode(user)
	if err != nil {
		logger.Error("error encoding user")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	user := &model.User{}
	if err := json.NewDecoder(r.Body).Decode(user); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	updatedSubscription, err := h.store.Update(r.Context(), user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(updatedSubscription)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := h.store.Delete(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
