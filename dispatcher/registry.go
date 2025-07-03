package dispatcher

import (
	"context"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/utils"
	"golang.org/x/sync/singleflight"
	"time"
)

type Registry struct {
	group singleflight.Group
	lru   *expirable.LRU[string, *Registration]
	db    *db.DB
}

func NewRegistry(db *db.DB) *Registry {
	return &Registry{
		lru: expirable.NewLRU[string, *Registration](100, nil, time.Second*60),
		db:  db,
	}
}

func (r *Registry) Warmup() error {
	workspaces, err := r.db.Workspaces.List(context.TODO(), &query.WorkspaceQuery{})
	if err != nil {
		return err
	}

	for _, w := range workspaces {
		v, err := r.load(context.TODO(), w.ID)
		if err != nil {
			return err
		}
		r.lru.Add(w.ID, v)
	}

	return nil
}

func (r *Registry) load(ctx context.Context, wid string) (*Registration, error) {
	var q query.EndpointQuery
	q.WorkspaceId = &wid
	q.Enabled = utils.Pointer(true)
	endpoints, err := r.db.Endpoints.List(ctx, &q)
	if err != nil {
		return nil, err
	}
	return NewRegistration(endpoints), nil
}

func (r *Registry) Unregister(wid string) {
	r.lru.Remove(wid)
}

func (r *Registry) LookUp(ctx context.Context, event *entities.Event) ([]*entities.Endpoint, error) {
	wid := event.WorkspaceId
	registration, exist := r.lru.Get(wid)
	if !exist {
		_, err, _ := r.group.Do(wid, func() (interface{}, error) {
			v, err := r.load(ctx, wid)
			if err != nil {
				return nil, err
			}
			r.lru.Add(wid, v)
			return nil, nil
		})

		if err != nil {
			return nil, err
		}

		registration, _ = r.lru.Get(wid)
	}

	if registration == nil {
		return nil, nil
	}

	return registration.LookUp(event), nil
}
