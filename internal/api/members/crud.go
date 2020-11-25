package members

import (
	"net/http"

	"demodesk/neko/internal/utils"
	"demodesk/neko/internal/types"
)

type MemberCreatePayload struct {
	ID string `json:"id"`
}

type MemberDataPayload struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	IsAdmin          bool   `json:"is_admin"`
	//Enabled          bool   `json:"enabled"`
	//CanControl       bool   `json:"can_control"`
	//CanWatch         bool   `json:"can_watch"`
	//ClipboardAccess  bool   `json:"clipboard_access"`
}

func (h *MembersHandler) membersCreate(w http.ResponseWriter, r *http.Request) {
	data := &MemberDataPayload{}
	if !utils.HttpJsonRequest(w, r, data) {
		return
	}

	session, err := h.sessions.Create(types.MemberProfile{
		Name: data.Name,
		IsAdmin: data.IsAdmin,
	})
	if err != nil {
		utils.HttpInternalServer(w, err)
		return
	}

	utils.HttpSuccess(w, MemberCreatePayload{
		ID: session.ID(),
	})
}

func (h *MembersHandler) membersRead(w http.ResponseWriter, r *http.Request) {
	data := &MemberDataPayload{}
	if !utils.HttpJsonRequest(w, r, data) {
		return
	}

	member := GetMember(r)

	utils.HttpSuccess(w, MemberDataPayload{
		ID: member.ID(),
		Name: member.Name(),
		IsAdmin: member.Admin(),
	})
}

func (h *MembersHandler) membersUpdate(w http.ResponseWriter, r *http.Request) {
	data := &MemberDataPayload{}
	if !utils.HttpJsonRequest(w, r, data) {
		return
	}

	member := GetMember(r)

	utils.HttpSuccess(w, MemberDataPayload{
		ID: member.ID(),
		Name: member.Name(),
		IsAdmin: member.Admin(),
	})
}

func (h *MembersHandler) membersDelete(w http.ResponseWriter, r *http.Request) {
	member := GetMember(r)

	if err := h.sessions.Delete(member.ID()); err != nil {
		utils.HttpInternalServer(w, err)
		return
	}

	utils.HttpSuccess(w)
}
