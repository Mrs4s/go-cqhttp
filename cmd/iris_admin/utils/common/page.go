package common

import (
	"github.com/GoAdminGroup/go-admin/template/types"
)

// PageSlice 对slice 进行分页式的截取
func PageSlice(sources []map[string]types.InfoItem, page, pagesize int) []map[string]types.InfoItem {
	var result []map[string]types.InfoItem
	lenth := len(sources)
	if pagesize > 0 {
		if pagesize*page <= lenth {
			result = sources[pagesize*(page-1) : pagesize*page]
		}
		if page-1 == 0 {
			result = sources
		}
		if pagesize*page > lenth {
			result = sources[pagesize*(page-1):]
		}
		return result
	}
	return result
}
