package handlers

import (
	"net/http"

	"github.com/gorilla/mux"

	membersv1 "github.com/sangay/web303/proto/members/v1"
	"github.com/sangay/web303/gateway/internal/clients"
	"github.com/sangay/web303/gateway/internal/httpjson"
	"github.com/sangay/web303/gateway/internal/statusmap"
)

// MembersHandler proxies REST calls for /members to Service B over gRPC.
type MembersHandler struct {
	clients *clients.Clients
}

func NewMembersHandler(c *clients.Clients) *MembersHandler {
	return &MembersHandler{clients: c}
}

// Register attaches all /members routes to the router.
func (h *MembersHandler) Register(r *mux.Router) {
	r.HandleFunc("/members", h.Create).Methods(http.MethodPost)
	r.HandleFunc("/members", h.List).Methods(http.MethodGet)
	r.HandleFunc("/members/{id}", h.Get).Methods(http.MethodGet)
	r.HandleFunc("/members/{id}", h.Update).Methods(http.MethodPut)
	r.HandleFunc("/members/{id}", h.Delete).Methods(http.MethodDelete)
}

// ---------- request/response DTOs (JSON shapes) ----------

type memberJSON struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Plan     string `json:"plan"`
	JoinedAt string `json:"joined_at,omitempty"`
}

// ---------- handlers ----------

func (h *MembersHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body memberJSON
	if err := httpjson.ReadJSON(r, &body); err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	resp, err := h.clients.Members.CreateMember(r.Context(), &membersv1.CreateMemberRequest{
		Name:  body.Name,
		Email: body.Email,
		Phone: body.Phone,
		Plan:  body.Plan,
	})
	if err != nil {
		code, msg := statusmap.ToHTTP(err)
		httpjson.WriteError(w, code, msg)
		return
	}
	httpjson.WriteJSON(w, http.StatusCreated, toMemberJSON(resp))
}

func (h *MembersHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	resp, err := h.clients.Members.GetMember(r.Context(), &membersv1.GetMemberRequest{Id: id})
	if err != nil {
		code, msg := statusmap.ToHTTP(err)
		httpjson.WriteError(w, code, msg)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, toMemberJSON(resp))
}

func (h *MembersHandler) List(w http.ResponseWriter, r *http.Request) {
	resp, err := h.clients.Members.ListMembers(r.Context(), &membersv1.ListMembersRequest{
		PageSize:  20,
		PageToken: r.URL.Query().Get("page_token"),
	})
	if err != nil {
		code, msg := statusmap.ToHTTP(err)
		httpjson.WriteError(w, code, msg)
		return
	}
	out := make([]memberJSON, 0, len(resp.Members))
	for _, m := range resp.Members {
		out = append(out, *toMemberJSON(m))
	}
	httpjson.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"members":         out,
		"next_page_token": resp.NextPageToken,
	})
}

func (h *MembersHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var body memberJSON
	if err := httpjson.ReadJSON(r, &body); err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	resp, err := h.clients.Members.UpdateMember(r.Context(), &membersv1.UpdateMemberRequest{
		Id:    id,
		Name:  body.Name,
		Email: body.Email,
		Phone: body.Phone,
		Plan:  body.Plan,
	})
	if err != nil {
		code, msg := statusmap.ToHTTP(err)
		httpjson.WriteError(w, code, msg)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, toMemberJSON(resp))
}

func (h *MembersHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	_, err := h.clients.Members.DeleteMember(r.Context(), &membersv1.DeleteMemberRequest{Id: id})
	if err != nil {
		code, msg := statusmap.ToHTTP(err)
		httpjson.WriteError(w, code, msg)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// toMemberJSON converts a proto Member to the JSON DTO.
func toMemberJSON(m *membersv1.Member) *memberJSON {
	if m == nil {
		return nil
	}
	out := &memberJSON{
		ID:    m.Id,
		Name:  m.Name,
		Email: m.Email,
		Phone: m.Phone,
		Plan:  m.Plan,
	}
	if m.JoinedAt != nil {
		out.JoinedAt = m.JoinedAt.AsTime().UTC().Format("2006-01-02T15:04:05Z")
	}
	return out
}
