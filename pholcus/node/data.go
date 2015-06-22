// 节点间交互数据结构定义。
package node

type Data struct {
	Type int
	Body interface{}
	From string
	To   string
}
