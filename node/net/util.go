// socket通信
package net

import (
	"encoding/json"
	"github.com/henrylee2cn/pholcus/runtime/cache"
	"log"
)

//编码通信数据
func marshal(data *cache.NetData) ([]byte, error) {
	b, err := json.Marshal(*data)
	//[]byte("}\r\n")==[]byte{125,13,10}
	// b = append(b, []byte{13, 10}...)
	return b, err
}

//解码通信数据
func unmarshal(data []byte) (*cache.NetData, error) {
	//[]byte("}\r\n")==[]byte{125,13,10}
	// for k, v := range data {
	// 	if v == byte(10) && data[k-1] == byte(13) && data[k-2] == byte(125) {
	// 		data = data[:k]
	// 		break
	// 	}
	// }
	d := new(cache.NetData)
	err := json.Unmarshal(data, d)
	return d, err
}

func checkError(err error) {
	if err != nil {
		log.Printf("Fatal error: %s", err.Error())
	}
}
