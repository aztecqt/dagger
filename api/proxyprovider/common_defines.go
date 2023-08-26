/*
 * @Author: aztec
 * @Date: 2023-05-25
 * @Description:
 */

package proxyprovider

import "time"

// 协议定义
type Proxy struct {
	Provider   string    `json:"src"`      // 供应商名称
	OrderID    int       `json:"order_id"` // 对应的订单ID，可以一个订单ID对应多个IP
	IP         string    `json:"ip"`
	Port       int       `json:"port_socks5"`
	UserName   string    `json:"username"`
	Password   string    `json:"password"`
	Location   string    `json:"loc"`
	CreateTime time.Time `json:"time_create"`
	ExpireTime time.Time `json:"time_expire"`
}

func (p *Proxy) Merge(other *Proxy) {
	*p = *other
}

type Provider interface {
	Name() string
	GetAllProxies() ([]Proxy, error)                                       // 取得所有可用的Proxy
	NewProxies(location string, period int, quantity int) ([]Proxy, error) // 下单
	RenewProxies(ids []int, period int) ([]Proxy, error)                   // 续费
	GetBalance() (float64, error)                                          // 获得当前余额
}
