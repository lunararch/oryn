package httpapi

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/lunararch/oryn/apps/server/internal/api"
	"github.com/lunararch/oryn/apps/server/internal/collections"
)

func (o Options) listCollections(w http.ResponseWriter, r *http.Request) {
	if o.Collections == nil {
		writeError(w, http.StatusServiceUnavailable, "library_not_configured", "collection service is not configured")
		return
	}
	items, err := o.Collections.List(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, api.CollectionsResponse{Items: collectionsDTO(items)})
}

func (o Options) getCollection(w http.ResponseWriter, r *http.Request) {
	if o.Collections == nil {
		writeError(w, http.StatusServiceUnavailable, "library_not_configured", "collection service is not configured")
		return
	}
	item, err := o.Collections.Get(r.Context(), chi.URLParam(r, "collectionID"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toCollectionDTO(item))
}

func (o Options) createCollection(w http.ResponseWriter, r *http.Request) {
	if o.Collections == nil {
		writeError(w, http.StatusServiceUnavailable, "library_not_configured", "collection service is not configured")
		return
	}
	var request api.CollectionRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	item, err := o.Collections.Create(r.Context(), request.Name, request.Description)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toCollectionDTO(item))
}

func (o Options) updateCollection(w http.ResponseWriter, r *http.Request) {
	if o.Collections == nil {
		writeError(w, http.StatusServiceUnavailable, "library_not_configured", "collection service is not configured")
		return
	}
	var request api.CollectionRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	item, err := o.Collections.Update(r.Context(), chi.URLParam(r, "collectionID"), request.Name, request.Description)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toCollectionDTO(item))
}

func (o Options) deleteCollection(w http.ResponseWriter, r *http.Request) {
	if o.Collections == nil {
		writeError(w, http.StatusServiceUnavailable, "library_not_configured", "collection service is not configured")
		return
	}
	if err := o.Collections.Delete(r.Context(), chi.URLParam(r, "collectionID")); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (o Options) listCollectionsForGame(w http.ResponseWriter, r *http.Request) {
	if o.Collections == nil {
		writeError(w, http.StatusServiceUnavailable, "library_not_configured", "collection service is not configured")
		return
	}
	items, err := o.Collections.ListForGame(r.Context(), chi.URLParam(r, "gameID"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]api.CollectionMembership, 0, len(items))
	for _, item := range items {
		out = append(out, api.CollectionMembership{
			ID:          item.ID,
			Name:        item.Name,
			Description: item.Description,
			GameCount:   item.GameCount,
			Contains:    item.Contains,
		})
	}
	writeJSON(w, http.StatusOK, api.CollectionMembershipsResponse{Items: out})
}

func (o Options) addGameToCollection(w http.ResponseWriter, r *http.Request) {
	if o.Collections == nil {
		writeError(w, http.StatusServiceUnavailable, "library_not_configured", "collection service is not configured")
		return
	}
	if err := o.Collections.AddGame(r.Context(), chi.URLParam(r, "collectionID"), chi.URLParam(r, "gameID")); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (o Options) removeGameFromCollection(w http.ResponseWriter, r *http.Request) {
	if o.Collections == nil {
		writeError(w, http.StatusServiceUnavailable, "library_not_configured", "collection service is not configured")
		return
	}
	if err := o.Collections.RemoveGame(r.Context(), chi.URLParam(r, "collectionID"), chi.URLParam(r, "gameID")); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func collectionsDTO(items []collections.Collection) []api.Collection {
	out := make([]api.Collection, 0, len(items))
	for _, item := range items {
		out = append(out, toCollectionDTO(item))
	}
	return out
}

func toCollectionDTO(item collections.Collection) api.Collection {
	return api.Collection{
		ID:          item.ID,
		Name:        item.Name,
		Description: item.Description,
		GameCount:   item.GameCount,
	}
}

var _ = errors.Is
