package collecting

import (
	"testing"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/internal/app/observer"
)

func TestChainCollector_Collect(t *testing.T) {
	type E struct {
		origin insolar.ID
		kind   string
		proper bool
	}
	newCollector := func() *ChainCollector {
		return NewChainCollector(
			&RelationDesc{
				Is: func(e interface{}) bool {
					return e.(E).kind == "parent"
				},
				Origin: func(e interface{}) insolar.ID {
					return e.(E).origin
				},
				Proper: func(e interface{}) bool {
					return e.(E).proper
				},
			},
			&RelationDesc{
				Is: func(e interface{}) bool {
					return e.(E).kind == "child"
				},
				Origin: func(e interface{}) insolar.ID {
					return e.(E).origin
				},
				Proper: func(e interface{}) bool {
					return e.(E).proper
				},
			},
		)
	}

	ids := []insolar.ID{gen.ID(), gen.ID(), gen.ID(), gen.ID()}

	table := []struct {
		name     string
		stream   []E
		expected []*observer.Chain
		checks   func(t *testing.T, collector *ChainCollector)
	}{
		{
			name:   "not ours",
			stream: []E{{kind: "some"}},
			expected: []*observer.Chain{},
			checks: func(t *testing.T, collector *ChainCollector) {
				require.Empty(t, collector.parents)
			},
		},
		{
			name:   "not match",
			stream: []E{
				{kind: "parent", origin: gen.ID()},
				{kind: "some", origin: gen.ID()},
			},
			expected: []*observer.Chain{},
			checks: func(t *testing.T, collector *ChainCollector) {
				require.Len(t, collector.parents, 1)
			},
		},
		{
			name:   "child-paren pairs",
			stream: []E{
				{kind: "parent", origin: ids[0], proper: true},
				{kind: "parent", origin: ids[1], proper: true},
				{kind: "parent", origin: ids[2], proper: false},
				{kind: "parent", origin: ids[3], proper: false},

				{kind: "child", origin: ids[0], proper: true},
				{kind: "child", origin: ids[1], proper: false},
				{kind: "child", origin: ids[2], proper: true},
				{kind: "child", origin: ids[3], proper: false},
			},
			expected: []*observer.Chain{
				{
					Parent: E{kind: "parent", origin: ids[0], proper: true},
					Child: E{kind: "child", origin: ids[0], proper: true},
				},
			},
			checks: func(t *testing.T, collector *ChainCollector) {
				require.Empty(t, collector.parents)
			},
		},
	}

	for _, test := range table {
		test := test
		t.Run(test.name, func(t *testing.T) {
			collector := newCollector()
			list := make([]*observer.Chain, 0)
			for _, e := range test.stream {
				res := collector.Collect(e)
				if res != nil {
					list = append(list, res)
				}
			}
			require.Equal(t, test.expected, list)
		})
	}
}
