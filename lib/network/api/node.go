package api

import (
	"net/http"

	"boscoin.io/sebak/lib/network/httputils"
)

func (api NetworkHandlerAPI) GetNodeHandler(w http.ResponseWriter, r *http.Request) {

	if err := httputils.WriteJSON(w, 200, api.localNode); err != nil {
		httputils.WriteJSONError(w, err)
		return
	}
}
