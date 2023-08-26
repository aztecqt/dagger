/*
 * @Author: aztec
 * @Date: 2023-05-25
 * @Description: proxyline的rest api，实现查询ip列表、购买、续费等操作
 */

package proxyprovider

import (
	"strings"
	"time"

	"github.com/aztecqt/dagger/util"
)

// 订单概要
type IpRoyalOrderBrief struct {
	Id           int    `json:"id"`
	Status       string `json:"status"`
	OrderDateStr string `json:"orderDate"`
	OrderDate    time.Time
}

func (o *IpRoyalOrderBrief) parse() {
	if t, err := time.Parse("2006-01-02 15:04:05", o.OrderDateStr); err == nil {
		o.OrderDate = t
	} else {
		panic(err)
	}
}

// 一个Proxy
type IpRoyalProxy struct {
	IP       string
	Port     int
	UserName string
	Password string
}

// 一个订单的信息
type IpRoyalOrder struct {
	Id             int    `json:"id"`
	Status         string `json:"status"` // confirmed=ok
	Location       string `json:"location"`
	ExpireDateStr  string `json:"expireDate"`
	ProductInfoStr string `json:"productInfo"`

	CreateDate time.Time
	ExpireDate time.Time
	Proxies    []IpRoyalProxy
}

func (i *IpRoyalOrder) parse(ob *IpRoyalOrderBrief) {
	if t, err := time.Parse("2006-01-02 15:04:05", i.ExpireDateStr); err == nil {
		i.ExpireDate = t
	} else {
		panic(err)
	}

	i.CreateDate = ob.OrderDate

	lines := strings.Split(i.ProductInfoStr, "\n")
	i.Proxies = make([]IpRoyalProxy, 0, len(lines))
	for _, line := range lines {
		ss := strings.Split(line, ":")
		if len(ss) == 4 {
			proxy := IpRoyalProxy{}
			proxy.IP = ss[0]
			proxy.Port = util.String2IntPanic(ss[1])
			proxy.UserName = ss[2]
			proxy.Password = ss[3]
			i.Proxies = append(i.Proxies, proxy)
		}
	}
}

func (i *IpRoyalOrder) toProxies(ob *IpRoyalOrderBrief) []Proxy {
	i.parse(ob)
	pxs := make([]Proxy, 0, len(i.Proxies))
	for _, irp := range i.Proxies {
		p := Proxy{}
		p.Provider = "iproyal"
		p.OrderID = i.Id
		p.IP = irp.IP
		p.Port = irp.Port
		p.UserName = irp.UserName
		p.Password = irp.Password
		p.Location = i.Location
		p.CreateTime = i.CreateDate
		p.ExpireTime = i.ExpireDate
		p.IP = irp.IP

		pxs = append(pxs, p)
	}

	return pxs
}
