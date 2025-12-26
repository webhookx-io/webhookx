package plugins

import (
	"context"
	"iter"
	"sort"
	"strings"
	"sync/atomic"

	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/plugin"
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

	indexes map[string][]plugin.Plugin
}

func NewIterator(version string) *Iterator {
	iterator := &Iterator{
		Version: version,
		indexes: make(map[string][]plugin.Plugin),
	}
	return iterator
}

func (it *Iterator) LoadPlugins(plugins []*entities.Plugin) error {
	indexes := it.indexes
	for _, plugin := range plugins {
		if !plugin.Enabled {
			continue
		}
		p, err := plugin.ToPlugin()
		if err != nil {
			return err
		}

		err = p.Init(plugin.Config)
		if err != nil {
			return err
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

	for index, plugins := range indexes {
		indexes[index] = it.sort(plugins)
	}

	return nil
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
