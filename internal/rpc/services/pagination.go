package services

import (
	"strconv"

	v1 "github.com/nickheyer/discordiance/pkg/proto/discordiance/v1"
)

const defaultPageSize = 25

func parsePagination(p *v1.PaginationRequest) (pageSize int, offset int) {
	pageSize = defaultPageSize
	if p != nil && p.PageSize > 0 {
		pageSize = int(p.PageSize)
	}
	if p != nil && p.PageToken != "" {
		offset, _ = strconv.Atoi(p.PageToken)
	}
	return
}

func buildPaginationResponse(offset, pageSize, total int) *v1.PaginationResponse {
	resp := &v1.PaginationResponse{
		TotalCount: int32(total),
	}
	if next := offset + pageSize; next < total {
		resp.NextPageToken = strconv.Itoa(next)
	}
	return resp
}
