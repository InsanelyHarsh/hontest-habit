package routes

import (
	"net/http"
	"strconv"

	"github.com/insanelyharsh/hontest-habit/internal/app/blocklist"
	"github.com/insanelyharsh/hontest-habit/internal/app/blocklist/models"
	"github.com/insanelyharsh/hontest-habit/internal/common/errors"
	"github.com/insanelyharsh/hontest-habit/internal/types"
	"github.com/insanelyharsh/hontest-habit/internal/webserver"
	"github.com/insanelyharsh/hontest-habit/internal/webserver/middlewares"
)

type BlockListController struct {
	blockListManager *blocklist.BlocklistManager
}

func NewBlocklistController(blockListManager *blocklist.BlocklistManager) *BlockListController {
	return &BlockListController{
		blockListManager: blockListManager,
	}
}

func (bc *BlockListController) Routes(group webserver.Group) {
	group.POST("/entries", bc.handleCreateEntry())
	group.GET("/entries", bc.handleGetEntries())
	group.DELETE("/entries/{id}", bc.handleRemoveEntry())
	group.POST("/entries/{id}/visits", bc.handleRecordVisit())
	group.GET("/entries/{id}/visits/remaining", bc.handleGetRemaining())
}

func (bc *BlockListController) handleCreateEntry() webserver.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		claims, ok := middlewares.ClaimsFromContext(r.Context())
		if !ok {
			return errors.Unauthorized("missing claims", nil)
		}

		userID, err := strconv.ParseInt(claims.Subject, 10, 64)
		if err != nil {
			return errors.Unauthorized("invalid claims", err)
		}

		var req models.CreateEntryRequest
		if err := webserver.DecodeJSON(r, &req); err != nil {
			return err
		}

		entry, err := bc.blockListManager.CreateEntry(r.Context(), types.UserId(userID), &req)
		if err != nil {
			return err
		}

		webserver.WriteJSON(w, http.StatusCreated, entry)
		return nil
	}
}

func (bc *BlockListController) handleGetEntries() webserver.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		claims, ok := middlewares.ClaimsFromContext(r.Context())
		if !ok {
			return errors.Unauthorized("missing claims", nil)
		}

		userID, err := strconv.ParseInt(claims.Subject, 10, 64)
		if err != nil {
			return errors.Unauthorized("invalid claims", err)
		}

		entries, err := bc.blockListManager.GetEntries(r.Context(), types.UserId(userID))
		if err != nil {
			return err
		}

		webserver.WriteJSON(w, http.StatusOK, entries)
		return nil
	}
}

func (bc *BlockListController) handleRemoveEntry() webserver.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		claims, ok := middlewares.ClaimsFromContext(r.Context())
		if !ok {
			return errors.Unauthorized("missing claims", nil)
		}

		userID, err := strconv.ParseInt(claims.Subject, 10, 64)
		if err != nil {
			return errors.Unauthorized("invalid claims", err)
		}

		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			return errors.BadRequest("invalid entry id", err)
		}

		if err := bc.blockListManager.RemoveEntry(r.Context(), types.UserId(userID), types.BlocklistId(id)); err != nil {
			return err
		}

		w.WriteHeader(http.StatusNoContent)
		return nil
	}
}

func (bc *BlockListController) handleRecordVisit() webserver.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		claims, ok := middlewares.ClaimsFromContext(r.Context())
		if !ok {
			return errors.Unauthorized("missing claims", nil)
		}

		userID, err := strconv.ParseInt(claims.Subject, 10, 64)
		if err != nil {
			return errors.Unauthorized("invalid claims", err)
		}

		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			return errors.BadRequest("invalid entry id", err)
		}

		counter, err := bc.blockListManager.RecordVisit(r.Context(), types.UserId(userID), types.BlocklistId(id))
		if err != nil {
			return err
		}

		webserver.WriteJSON(w, http.StatusCreated, counter)
		return nil
	}
}

func (bc *BlockListController) handleGetRemaining() webserver.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		claims, ok := middlewares.ClaimsFromContext(r.Context())
		if !ok {
			return errors.Unauthorized("missing claims", nil)
		}

		userID, err := strconv.ParseInt(claims.Subject, 10, 64)
		if err != nil {
			return errors.Unauthorized("invalid claims", err)
		}

		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			return errors.BadRequest("invalid entry id", err)
		}

		counter, err := bc.blockListManager.GetRemaining(r.Context(), types.UserId(userID), types.BlocklistId(id))
		if err != nil {
			return err
		}

		webserver.WriteJSON(w, http.StatusOK, counter)
		return nil
	}
}
