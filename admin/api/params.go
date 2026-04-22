package api

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/webhookx-io/webhookx/db/dao"
)

const (
	DefaultPageSize = 20
)

type Params interface {
	Validate() error
}

type ListParams struct {
	// Deprecated
	PageNo int `form:"page_no"`
	// Deprecated
	PageSize int `form:"page_size"`

	// limit parameter
	Limit *int `form:"limit"`
	// after parameter
	After *string `form:"after"`
	// before parameter
	Before *string `form:"before"`

	// sort parameter
	Sort string `form:"sort"`
}

func (p *ListParams) Validate() error {
	if p.After != nil && p.Before != nil {
		return errors.New("cannot pass both 'after' and 'before' params at the same time")
	}
	return nil
}

func (p *ListParams) Query() *dao.Query {
	var query dao.Query

	pageNo := max(p.PageNo, 1)
	pageSize := p.PageSize
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	query.Page(pageNo, pageSize)
	if p.Limit != nil {
		query.Page(1, *p.Limit)
	}

	sort := p.Sort
	if sort == "" {
		sort = "id.desc"
	}
	field, order, _ := strings.Cut(sort, ".")
	if p.After != nil {
		switch order {
		case "asc":
			query.Where("id", dao.GreaterThan, *p.After)
		case "desc":
			query.Where("id", dao.LessThan, *p.After)
		}
	}
	if p.Before != nil {
		switch order {
		case "asc":
			query.Where("id", dao.LessThan, *p.Before)
		case "desc":
			query.Where("id", dao.GreaterThan, *p.Before)
		}
	}

	if p.Before != nil { // reverse order
		if order == "asc" {
			order = "desc"
		} else if order == "desc" {
			order = "asc"
		}
		query.Reverse = true
	}
	query.Order(field, order)

	if p.Limit != nil || p.After != nil || p.Before != nil {
		query.CursorModel = true
	}
	return &query
}

type EndpointListParams struct {
	ListParams

	Name         *string           `form:"name"`
	Enabled      *bool             `form:"enabled"`
	CreatedAt    *int64            `form:"created_at"`
	CreatedAtGT  *int64            `form:"created_at[gt]"`
	CreatedAtGTE *int64            `form:"created_at[gte]"`
	CreatedAtLT  *int64            `form:"created_at[lt]"`
	CreatedAtLTE *int64            `form:"created_at[lte]"`
	Metadata     map[string]string `form:"metadata"`
}

func (p *EndpointListParams) Query() *dao.Query {
	query := p.ListParams.Query()

	if p.Name != nil {
		query.Where("name", dao.Equal, *p.Name)
	}
	if p.Enabled != nil {
		query.Where("enabled", dao.Equal, *p.Enabled)
	}
	if p.CreatedAt != nil {
		query.Where("created_at", dao.Equal, time.UnixMilli(*p.CreatedAt))
	}
	if p.CreatedAtGT != nil {
		query.Where("created_at", dao.GreaterThan, time.UnixMilli(*p.CreatedAtGT))
	}
	if p.CreatedAtGTE != nil {
		query.Where("created_at", dao.GreaterThanOrEqual, time.UnixMilli(*p.CreatedAtGTE))
	}
	if p.CreatedAtLT != nil {
		query.Where("created_at", dao.LessThan, time.UnixMilli(*p.CreatedAtLT))
	}
	if p.CreatedAtLTE != nil {
		query.Where("created_at", dao.LessThanOrEqual, time.UnixMilli(*p.CreatedAtLTE))
	}
	if len(p.Metadata) > 0 {
		b, _ := json.Marshal(p.Metadata)
		query.Where("metadata", dao.JsonContain, string(b))
	}
	return query
}

type SourceListParams struct {
	ListParams

	Name         *string           `form:"name"`
	Enabled      *bool             `form:"enabled"`
	CreatedAt    *int64            `form:"created_at"`
	CreatedAtGT  *int64            `form:"created_at[gt]"`
	CreatedAtGTE *int64            `form:"created_at[gte]"`
	CreatedAtLT  *int64            `form:"created_at[lt]"`
	CreatedAtLTE *int64            `form:"created_at[lte]"`
	Metadata     map[string]string `form:"metadata"`
}

func (p *SourceListParams) Query() *dao.Query {
	query := p.ListParams.Query()

	if p.Name != nil {
		query.Where("name", dao.Equal, *p.Name)
	}
	if p.Enabled != nil {
		query.Where("enabled", dao.Equal, *p.Enabled)
	}
	if p.CreatedAt != nil {
		query.Where("created_at", dao.Equal, time.UnixMilli(*p.CreatedAt))
	}
	if p.CreatedAtGT != nil {
		query.Where("created_at", dao.GreaterThan, time.UnixMilli(*p.CreatedAtGT))
	}
	if p.CreatedAtGTE != nil {
		query.Where("created_at", dao.GreaterThanOrEqual, time.UnixMilli(*p.CreatedAtGTE))
	}
	if p.CreatedAtLT != nil {
		query.Where("created_at", dao.LessThan, time.UnixMilli(*p.CreatedAtLT))
	}
	if p.CreatedAtLTE != nil {
		query.Where("created_at", dao.LessThanOrEqual, time.UnixMilli(*p.CreatedAtLTE))
	}
	if len(p.Metadata) > 0 {
		b, _ := json.Marshal(p.Metadata)
		query.Where("metadata", dao.JsonContain, string(b))
	}
	return query
}

type PluginListParams struct {
	ListParams

	Name         *string           `form:"name"`
	Enabled      *bool             `form:"enabled"`
	CreatedAt    *int64            `form:"created_at"`
	CreatedAtGT  *int64            `form:"created_at[gt]"`
	CreatedAtGTE *int64            `form:"created_at[gte]"`
	CreatedAtLT  *int64            `form:"created_at[lt]"`
	CreatedAtLTE *int64            `form:"created_at[lte]"`
	Metadata     map[string]string `form:"metadata"`
	EndpointId   *string           `form:"endpoint_id"`
	SourceId     *string           `form:"source_id"`
}

func (p *PluginListParams) Query() *dao.Query {
	query := p.ListParams.Query()

	if p.Name != nil {
		query.Where("name", dao.Equal, *p.Name)
	}
	if p.Enabled != nil {
		query.Where("enabled", dao.Equal, *p.Enabled)
	}
	if p.CreatedAt != nil {
		query.Where("created_at", dao.Equal, time.UnixMilli(*p.CreatedAt))
	}
	if p.CreatedAtGT != nil {
		query.Where("created_at", dao.GreaterThan, time.UnixMilli(*p.CreatedAtGT))
	}
	if p.CreatedAtGTE != nil {
		query.Where("created_at", dao.GreaterThanOrEqual, time.UnixMilli(*p.CreatedAtGTE))
	}
	if p.CreatedAtLT != nil {
		query.Where("created_at", dao.LessThan, time.UnixMilli(*p.CreatedAtLT))
	}
	if p.CreatedAtLTE != nil {
		query.Where("created_at", dao.LessThanOrEqual, time.UnixMilli(*p.CreatedAtLTE))
	}
	if len(p.Metadata) > 0 {
		b, _ := json.Marshal(p.Metadata)
		query.Where("metadata", dao.JsonContain, string(b))
	}
	if p.EndpointId != nil {
		query.Where("endpoint_id", dao.Equal, *p.EndpointId)
	}
	if p.SourceId != nil {
		query.Where("source_id", dao.Equal, *p.SourceId)
	}
	return query
}

type AttemptListParams struct {
	ListParams

	CreatedAt      *int64  `form:"created_at"`
	CreatedAtGT    *int64  `form:"created_at[gt]"`
	CreatedAtGTE   *int64  `form:"created_at[gte]"`
	CreatedAtLT    *int64  `form:"created_at[lt]"`
	CreatedAtLTE   *int64  `form:"created_at[lte]"`
	EventId        *string `form:"event_id"`
	EndpointId     *string `form:"endpoint_id"`
	Status         *string `form:"status"`
	AttemptedAt    *int64  `form:"attempted_at"`
	AttemptedAtGT  *int64  `form:"attempted_at[gt]"`
	AttemptedAtGTE *int64  `form:"attempted_at[gte]"`
	AttemptedAtLT  *int64  `form:"attempted_at[lt]"`
	AttemptedAtLTE *int64  `form:"attempted_at[lte]"`
}

func (p *AttemptListParams) Query() *dao.Query {
	query := p.ListParams.Query()

	if p.CreatedAt != nil {
		query.Where("created_at", dao.Equal, time.UnixMilli(*p.CreatedAt))
	}
	if p.CreatedAtGT != nil {
		query.Where("created_at", dao.GreaterThan, time.UnixMilli(*p.CreatedAtGT))
	}
	if p.CreatedAtGTE != nil {
		query.Where("created_at", dao.GreaterThanOrEqual, time.UnixMilli(*p.CreatedAtGTE))
	}
	if p.CreatedAtLT != nil {
		query.Where("created_at", dao.LessThan, time.UnixMilli(*p.CreatedAtLT))
	}
	if p.CreatedAtLTE != nil {
		query.Where("created_at", dao.LessThanOrEqual, time.UnixMilli(*p.CreatedAtLTE))
	}
	if p.EventId != nil {
		query.Where("event_id", dao.Equal, *p.EventId)
	}
	if p.EndpointId != nil {
		query.Where("endpoint_id", dao.Equal, *p.EndpointId)
	}
	if p.Status != nil {
		query.Where("status", dao.Equal, *p.Status)
	}
	if p.AttemptedAt != nil {
		query.Where("attempted_at", dao.Equal, time.UnixMilli(*p.AttemptedAt))
	}
	if p.AttemptedAtGT != nil {
		query.Where("attempted_at", dao.GreaterThan, time.UnixMilli(*p.AttemptedAtGT))
	}
	if p.AttemptedAtGTE != nil {
		query.Where("attempted_at", dao.GreaterThanOrEqual, time.UnixMilli(*p.AttemptedAtGTE))
	}
	if p.AttemptedAtLT != nil {
		query.Where("attempted_at", dao.LessThan, time.UnixMilli(*p.AttemptedAtLT))
	}
	if p.AttemptedAtLTE != nil {
		query.Where("attempted_at", dao.LessThanOrEqual, time.UnixMilli(*p.AttemptedAtLTE))
	}
	return query
}

type WorkspaceListParams struct {
	ListParams

	Name         *string           `form:"name"`
	CreatedAt    *int64            `form:"created_at"`
	CreatedAtGT  *int64            `form:"created_at[gt]"`
	CreatedAtGTE *int64            `form:"created_at[gte]"`
	CreatedAtLT  *int64            `form:"created_at[lt]"`
	CreatedAtLTE *int64            `form:"created_at[lte]"`
	Metadata     map[string]string `form:"metadata"`
}

func (p *WorkspaceListParams) Query() *dao.Query {
	query := p.ListParams.Query()

	if p.Name != nil {
		query.Where("name", dao.Equal, *p.Name)
	}
	if p.CreatedAt != nil {
		query.Where("created_at", dao.Equal, time.UnixMilli(*p.CreatedAt))
	}
	if p.CreatedAtGT != nil {
		query.Where("created_at", dao.GreaterThan, time.UnixMilli(*p.CreatedAtGT))
	}
	if p.CreatedAtGTE != nil {
		query.Where("created_at", dao.GreaterThanOrEqual, time.UnixMilli(*p.CreatedAtGTE))
	}
	if p.CreatedAtLT != nil {
		query.Where("created_at", dao.LessThan, time.UnixMilli(*p.CreatedAtLT))
	}
	if p.CreatedAtLTE != nil {
		query.Where("created_at", dao.LessThanOrEqual, time.UnixMilli(*p.CreatedAtLTE))
	}
	if len(p.Metadata) > 0 {
		b, _ := json.Marshal(p.Metadata)
		query.Where("metadata", dao.JsonContain, string(b))
	}
	return query
}

type EventListParams struct {
	ListParams

	CreatedAt     *int64  `form:"created_at"`
	CreatedAtGT   *int64  `form:"created_at[gt]"`
	CreatedAtGTE  *int64  `form:"created_at[gte]"`
	CreatedAtLT   *int64  `form:"created_at[lt]"`
	CreatedAtLTE  *int64  `form:"created_at[lte]"`
	EventType     *string `form:"event_type"`
	UniqueId      *string `form:"unique_id"`
	IngestedAt    *int64  `form:"ingested_at"`
	IngestedAtGT  *int64  `form:"ingested_at[gt]"`
	IngestedAtGTE *int64  `form:"ingested_at[gte]"`
	IngestedAtLT  *int64  `form:"ingested_at[lt]"`
	IngestedAtLTE *int64  `form:"ingested_at[lte]"`
}

func (p *EventListParams) Query() *dao.Query {
	query := p.ListParams.Query()

	if p.CreatedAt != nil {
		query.Where("created_at", dao.Equal, time.UnixMilli(*p.CreatedAt))
	}
	if p.CreatedAtGT != nil {
		query.Where("created_at", dao.GreaterThan, time.UnixMilli(*p.CreatedAtGT))
	}
	if p.CreatedAtGTE != nil {
		query.Where("created_at", dao.GreaterThanOrEqual, time.UnixMilli(*p.CreatedAtGTE))
	}
	if p.CreatedAtLT != nil {
		query.Where("created_at", dao.LessThan, time.UnixMilli(*p.CreatedAtLT))
	}
	if p.CreatedAtLTE != nil {
		query.Where("created_at", dao.LessThanOrEqual, time.UnixMilli(*p.CreatedAtLTE))
	}
	if p.EventType != nil {
		query.Where("event_type", dao.Equal, *p.EventType)
	}
	if p.UniqueId != nil {
		query.Where("unique_id", dao.Equal, *p.UniqueId)
	}
	if p.IngestedAt != nil {
		query.Where("ingested_at", dao.Equal, time.UnixMilli(*p.IngestedAt))
	}
	if p.IngestedAtGT != nil {
		query.Where("ingested_at", dao.GreaterThan, time.UnixMilli(*p.IngestedAtGT))
	}
	if p.IngestedAtGTE != nil {
		query.Where("ingested_at", dao.GreaterThanOrEqual, time.UnixMilli(*p.IngestedAtGTE))
	}
	if p.IngestedAtLT != nil {
		query.Where("ingested_at", dao.LessThan, time.UnixMilli(*p.IngestedAtLT))
	}
	if p.IngestedAtLTE != nil {
		query.Where("ingested_at", dao.LessThanOrEqual, time.UnixMilli(*p.IngestedAtLTE))
	}
	return query
}
