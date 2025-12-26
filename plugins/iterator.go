package plugins

import (
	"context"
	"fmt"
	"iter"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/pkg/secret"
	"github.com/webhookx-io/webhookx/pkg/secret/reference"
	"golang.org/x/sync/errgroup"
)

var instance atomic.Pointer[Iterator]

func init() {
	instance.Store(NewIterator(""))
}

func LoadIterator() *Iterator {
	return instance.Load()
}

func SetIterator(it *Iterator) {
	instance.Store(it)
}

type Phase string

const (
	PhaseInbound  Phase = "inbound"
	PhaseOutbound Phase = "outbound"
)

type Iterator struct {
	Version string
	Created time.Time

	sm      *secret.SecretManager
	indexes map[string][]plugin.Plugin
}

func NewIterator(version string) *Iterator {
	iterator := &Iterator{
		Version: version,
		Created: time.Now(),
		indexes: make(map[string][]plugin.Plugin),
	}
	return iterator
}

func (it *Iterator) LoadPlugins(plugins []*entities.Plugin) error {
	indexes := it.indexes
	group, ctx := errgroup.WithContext(context.TODO())
	for _, plugin := range plugins {
		if !plugin.Enabled {
			continue
		}
		p, err := plugin.ToPlugin()
		if err != nil {
			return err
		}

		// resolve references
		if it.sm != nil {
			group.Go(func() error {
				_, err := resolveReference(ctx, it.sm, map[string]interface{}(plugin.Config), nil)
				if err != nil {
					return fmt.Errorf("plugin{id=%s} configuration reference resolve failed: %w", plugin.ID, err)
				}
				if err := p.Init(plugin.Config); err != nil {
					return fmt.Errorf("plugin{id=%s} configuration init failed: %w", plugin.ID, err)
				}
				return nil
			})
		} else {
			if err := p.Init(plugin.Config); err != nil {
				return fmt.Errorf("plugin{id=%s} configuration init failed: %w", plugin.ID, err)
			}
		}

		if plugin.SourceId != nil {
			index := it.index(PhaseInbound, *plugin.SourceId)
			indexes[index] = append(indexes[index], p)
		}
		if plugin.EndpointId != nil {
			index := it.index(PhaseOutbound, *plugin.EndpointId)
			indexes[index] = append(indexes[index], p)
		}
	}
	if err := group.Wait(); err != nil {
		return err
	}

	for index, plugins := range indexes {
		indexes[index] = it.sort(plugins)
	}

	return nil
}

func resolveReference(ctx context.Context, sm *secret.SecretManager, value interface{}, paths []string) (interface{}, error) {
	switch val := value.(type) {
	case map[string]interface{}:
		for k, v := range val {
			resolved, err := resolveReference(ctx, sm, v, append(paths, k))
			if err != nil {
				return nil, err
			}
			val[k] = resolved
		}
		return val, nil
	case []interface{}:
		for i, v := range val {
			resolved, err := resolveReference(ctx, sm, v, append(paths, fmt.Sprintf("[%d]", i)))
			if err != nil {
				return nil, err
			}
			val[i] = resolved
		}
		return val, nil
	case string:
		if reference.IsReference(val) {
			ref, err := reference.Parse(val)
			if err != nil {
				return nil, fmt.Errorf("property %q parse error: %w", strings.Join(paths, "."), err)
			}
			resolved, err := sm.ResolveReference(ctx, ref)
			if err != nil {
				return nil, fmt.Errorf("property %q resolve error: %w", strings.Join(paths, "."), err)
			}
			return resolved, nil
		}
		return val, nil
	default:
		return val, nil
	}
}

func (it *Iterator) WithSecretManager(sm *secret.SecretManager) {
	it.sm = sm
}

func (it *Iterator) sort(plugins []plugin.Plugin) []plugin.Plugin {
	sort.Slice(plugins, func(i, j int) bool {
		pi := plugins[i]
		pj := plugins[j]
		if pi.Priority() == pj.Priority() {
			return pi.Name() > pj.Name()
		}
		return pi.Priority() > pj.Priority()
	})
	return plugins
}

func (it *Iterator) index(phase Phase, id string) string {
	sb := strings.Builder{}
	sb.WriteString(string(phase))
	sb.WriteString(":")
	sb.WriteString(id)
	return sb.String()
}

func (it *Iterator) Iterate(ctx context.Context, phase Phase, id string) iter.Seq[plugin.Plugin] {
	index := it.index(phase, id)
	plugins, exist := it.indexes[index]
	if !exist {
		return func(yield func(plugin.Plugin) bool) {}
	}
	return func(yield func(p plugin.Plugin) bool) {
		for _, v := range plugins {
			if !yield(v) {
				return
			}
		}
	}
}
