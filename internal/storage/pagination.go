package storage

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/hnzhou16/project-cocraft-server/internal/security"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PaginationQuery struct {
	Limit         int                  `json:"limit,omitempty" validate:"gte=1,lte=20"`
	Offset        int                  `json:"offset,omitempty" validate:"gte=0"`
	Sort          string               `json:"sort,omitempty" validate:"oneof=asc desc"`
	ShowFollowing bool                 `json:"show_following,omitempty"`
	FolloweeIDs   []primitive.ObjectID `json:"followee_ids"`
	ShowMentioned bool                 `json:"show_mentioned,omitempty"`
	Roles         []security.Role      `json:"roles,omitempty" validate:"valid_roles_slice"`
}

func (pq *PaginationQuery) Parse(r *http.Request) error {
	q := r.URL.Query()

	log.Println(q)

	if limit, err := strconv.Atoi(q.Get("limit")); err == nil && limit > 0 {
		pq.Limit = limit
	}

	if offset, err := strconv.Atoi(q.Get("offset")); err == nil && offset > 0 {
		pq.Offset = offset
	}

	sort := q.Get("sort")
	if sort == "asc" {
		pq.Sort = sort
	}

	pq.ShowFollowing = q.Get("following") == "true"
	pq.ShowMentioned = q.Get("mentioned") == "true"

	if rolesStr := q.Get("roles"); rolesStr != "" && rolesStr != "undefined" {
		rolesStrSlice := strings.Split(rolesStr, ",")
		for _, roleStr := range rolesStrSlice {
			pq.Roles = append(pq.Roles, security.Role(roleStr))
		}
	}

	log.Println(pq)

	return nil
}
