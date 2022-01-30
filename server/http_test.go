package server

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestHttpCtx_Get(t *testing.T) {
	cases := []struct {
		ctx      *httpCtx
		key      string
		expected string
	}{
		{
			ctx: &httpCtx{
				json: gjson.Result{},
				query: url.Values{
					"sub_type": []string{"hello"},
					"type":     []string{"world"},
				},
			},
			key:      "[sub_type,type].0",
			expected: "hello",
		},
		{
			ctx: &httpCtx{
				json: gjson.Result{},
				query: url.Values{
					"type": []string{"114514"},
				},
			},
			key:      "[sub_type,type].0",
			expected: "114514",
		},
	}
	for _, c := range cases {
		assert.Equal(t, c.expected, c.ctx.Get(c.key).String())
	}
}
