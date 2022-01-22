package tables

import "github.com/GoAdminGroup/go-admin/plugins/admin/modules/table"

// generators is a map of table models.
//
// The key of generators is the prefix of table info url.
// The corresponding value is the Form and TableName data.
//
// http://{{config.Domain}}:{{Port}}/{{config.Prefix}}/info/{{key}}
//
// example:
//
// "users"   => http://localhost:8080/admin/info/users
// "posts"   => http://localhost:8080/admin/info/posts
// "authors" => http://localhost:8080/admin/info/authors

// Generators 初始化
var Generators = map[string]table.Generator{
	"qq_config": GetQQConfigTable,
}
