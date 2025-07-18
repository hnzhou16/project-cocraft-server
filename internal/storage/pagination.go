package storage

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/hnzhou16/project-cocraft-server/internal/security"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CursorQuery struct {
	Limit         int                  `json:"limit,omitempty" validate:"gte=1,lte=20"`
	Cursor        string               `json:"cursor,omitempty"`
	Sort          string               `json:"sort,omitempty" validate:"oneof=asc desc"`
	ShowFollowing bool                 `json:"show_following,omitempty"`
	FolloweeIDs   []primitive.ObjectID `json:"followee_ids"`
	ShowMentioned bool                 `json:"show_mentioned,omitempty"`
	Roles         []security.Role      `json:"roles,omitempty" validate:"valid_roles_slice"`
	Search        string               `json:"search,omitempty"`
}

func (cq *CursorQuery) Parse(r *http.Request) error {
	q := r.URL.Query()

	if limit, err := strconv.Atoi(q.Get("limit")); err == nil && limit > 0 {
		cq.Limit = limit
	}

	if cursor := q.Get("cursor"); cursor != "" && cursor != "undefined" {
		cq.Cursor = cursor
	}

	sort := q.Get("sort")
	if sort == "asc" {
		cq.Sort = sort
	}

	cq.ShowFollowing = q.Get("following") == "true"
	cq.ShowMentioned = q.Get("mentioned") == "true"

	if rolesStr := q.Get("roles"); rolesStr != "" && rolesStr != "undefined" {
		rolesStrSlice := strings.Split(rolesStr, ",")
		for _, roleStr := range rolesStrSlice {
			cq.Roles = append(cq.Roles, security.Role(roleStr))
		}
	}

	if search := q.Get("search"); search != "" && search != "undefined" {
		cq.Search = search
	}

	return nil
}
