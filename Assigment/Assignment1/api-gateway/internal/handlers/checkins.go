package handlers

import (
	"net/http"

	"github.com/gorilla/mux"

	checkinsv1 "github.com/sangay/web303/proto/checkins/v1"
	"github.com/sangay/web303/gateway/internal/clients"
	"github.com/sangay/web303/gateway/internal/httpjson"
	"github.com/sangay/web303/gateway/internal/statusmap"
)

// CheckInsHandler proxies REST calls for /checkins to Service A over gRPC.
type CheckInsHandler struct {
	clients *clients.Clients
}

func NewCheckInsHandler(c *clients.Clients) *CheckInsHandler {
	return &CheckInsHandler{clients: c}
}

func (h *CheckInsHandler) Register(r *mux.Router) {
	r.HandleFunc("/checkins", h.Create).Methods(http.MethodPost)
	r.HandleFunc("/checkins", h.List).Methods(http.MethodGet)
	r.HandleFunc("/checkins/{id}", h.Get).Methods(http.MethodGet)
	r.HandleFunc("/checkins/{id}", h.Update).Methods(http.MethodPut)
	r.HandleFunc("/checkins/{id}", h.Delete).Methods(http.MethodDelete)
}

// ---------- DTOs ----------

type checkInJSON struct {
	ID         string `json:"id,omitempty"`
	MemberID   string `json:"member_id"`
	CheckInAt  string `json:"check_in_at,omitempty"`
	Notes      string `json:"notes,omitempty"`
}

// ---------- handlers ----------

func (h *CheckInsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body checkInJSON
	if err := httpjson.ReadJSON(r, &body); err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	resp, err := h.clients.CheckIns.CreateCheckIn(r.Context(), &checkinsv1.CreateCheckInRequest{
		MemberId: body.MemberID,
		Notes:    body.Notes,
	})
	if err != nil {
		code, msg := statusmap.ToHTTP(err)
		httpjson.WriteError(w, code, msg)
		return
	}
	httpjson.WriteJSON(w, http.StatusCreated, toCheckInJSON(resp))
}

func (h *CheckInsHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	resp, err := h.clients.CheckIns.GetCheckIn(r.Context(), &checkinsv1.GetCheckInRequest{Id: id})
	if err != nil {
		code, msg := statusmap.ToHTTP(err)
		httpjson.WriteError(w, code, msg)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, toCheckInJSON(resp))
}

func (h *CheckInsHandler) List(w http.ResponseWriter, r *http.Request) {
	resp, err := h.clients.CheckIns.ListCheckIns(r.Context(), &checkinsv1.ListCheckInsRequest{
		MemberId:  r.URL.Query().Get("member_id"),
		PageSize:  20,
		PageToken: r.URL.Query().Get("page_token"),
	})
	if err != nil {
		code, msg := statusmap.ToHTTP(err)
		httpjson.WriteError(w, code, msg)
		return
	}
	out := make([]checkInJSON, 0, len(resp.Checkins))
	for _, c := range resp.Checkins {
		out = append(out, *toCheckInJSON(c))
	}
	httpjson.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"checkins":        out,
		"next_page_token": resp.NextPageToken,
	})
}

func (h *CheckInsHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var body checkInJSON
	if err := httpjson.ReadJSON(r, &body); err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	resp, err := h.clients.CheckIns.UpdateCheckIn(r.Context(), &checkinsv1.UpdateCheckInRequest{
		Id:    id,
		Notes: body.Notes,
	})
	if err != nil {
		code, msg := statusmap.ToHTTP(err)
		httpjson.WriteError(w, code, msg)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, toCheckInJSON(resp))
}

func (h *CheckInsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	_, err := h.clients.CheckIns.DeleteCheckIn(r.Context(), &checkinsv1.DeleteCheckInRequest{Id: id})
	if err != nil {
		code, msg := statusmap.ToHTTP(err)
		httpjson.WriteError(w, code, msg)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func toCheckInJSON(c *checkinsv1.CheckIn) *checkInJSON {
	if c == nil {
		return nil
	}
	out := &checkInJSON{
		ID:       c.Id,
		MemberID: c.MemberId,
		Notes:    c.Notes,
	}
	if c.CheckInAt != nil {
		out.CheckInAt = c.CheckInAt.AsTime().UTC().Format("2006-01-02T15:04:05Z")
	}
	return out
}
