/*
 * @Author: aztec
 * @Date: 2023-05-25
 * @Description: iproyal的api实现
 */

package proxyprovider

import (
	"errors"
	"fmt"

	"aztecqt/dagger/util"
	"aztecqt/dagger/util/network"
)

type IpRoyalApi struct {
	rootUrl   string
	token     string
	logPrefix string
	headers   map[string]string
}

func (p *IpRoyalApi) Init() {
	p.rootUrl = "https://dashboard.iproyal.com/api/servers/proxies/reseller/"
	p.token = "LhsXK5p2zXFWVyTIiEtucfhFf2O8QXLOUoouW3GsflilWeJRyoJrZpKsqZpa"
	p.logPrefix = "IpRoyal"

	p.headers = make(map[string]string)
	p.headers["X-Access-Token"] = fmt.Sprintf("Bearer %s", p.token)
	p.headers["Content-Type"] = "application/json"
}

func (p *IpRoyalApi) Name() string {
	return "iproyal"
}

func (p *IpRoyalApi) GetAllProxies() (pxs []Proxy, err error) {
	defer util.DefaultRecoverWithCallback(func(estr string) {
		err = errors.New(estr)
	})

	pxs = make([]Proxy, 0)

	// 先拉取订单列表
	url := fmt.Sprintf("%sorders", p.rootUrl)
	respOrders, err := network.ParseHttpResult[[]IpRoyalOrderBrief](p.logPrefix, "Orders", url, "GET", "", p.headers, nil, nil)
	if err != nil {
		return
	}

	// 然后拉取所有confirmed状态的订单的详情
	for _, ob := range *respOrders {
		ob.parse()
		if ob.Status == "confirmed" {
			url = fmt.Sprintf("%s%d/order?socks5_port=1", p.rootUrl, ob.Id)
			if respOrder, e := network.ParseHttpResult[IpRoyalOrder](p.logPrefix, "Order", url, "GET", "", p.headers, nil, nil); e == nil {
				pxs_temp := respOrder.toProxies(&ob)
				pxs = append(pxs, pxs_temp...)
			} else {
				err = e
				return
			}
		}
	}

	return
}

func (p *IpRoyalApi) RenewProxies(ids []int, period int) (pxs []Proxy, err error) {
	defer util.DefaultRecoverWithCallback(func(estr string) {
		err = errors.New(estr)
	})

	pxs = make([]Proxy, 0)

	if len(ids) == 0 {
		return nil, errors.New("missing id")
	}

	// 先拉取订单列表
	url := fmt.Sprintf("%sorders", p.rootUrl)
	respOrders, err := network.ParseHttpResult[[]IpRoyalOrderBrief](p.logPrefix, "Orders", url, "GET", "", p.headers, nil, nil)
	if err != nil {
		return
	}

	dict := make(map[int]IpRoyalOrderBrief)
	for _, ob := range *respOrders {
		dict[ob.Id] = ob
	}

	// 逐个遍历订单并续费
	for _, id := range ids {
		if ob, ok := dict[id]; ok {
			url = fmt.Sprintf("%s%d/extend-order", p.rootUrl, id)
			if respOrder, e := network.ParseHttpResult[IpRoyalOrder](p.logPrefix, "Order", url, "GET", "", p.headers, nil, nil); e == nil {
				pxs_temp := respOrder.toProxies(&ob)
				pxs = append(pxs, pxs_temp...)
			} else {
				err = e
				return
			}
		} else {
			err = errors.New(fmt.Sprintf("unknown order id:%d", id))
		}
	}

	return
}

func (p *IpRoyalApi) NewProxies(location string, period int, quantity int) (pxs []Proxy, err error) {
	return nil, errors.New("no impl")
}

func (p *IpRoyalApi) GetBalance() (bal float64, err error) {
	return -1, errors.New("no impl")
}
