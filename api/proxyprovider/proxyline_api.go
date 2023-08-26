/*
 * @Author: aztec
 * @Date: 2023-05-23
 * @Description: proxyline的rest api，实现查询ip列表、购买、续费等操作
 */

package proxyprovider

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"aztecqt/dagger/util"
	"aztecqt/dagger/util/network"
)

// 实现Provider接口
type ProxyLineApi struct {
	rootUrl   string
	apiKey    string
	logPrefix string
}

func (p *ProxyLineApi) Init() {
	p.rootUrl = "https://panel.proxyline.net/api/"
	p.apiKey = "XWr0f4mzFvJArtpLluGPPuW5BrrS2bEWG0MBGn9I"
	p.logPrefix = "proxyline"
}

func (p *ProxyLineApi) Name() string {
	return "proxyline"
}

func (p *ProxyLineApi) GetAllProxies() (pxs []Proxy, err error) {
	defer util.DefaultRecoverWithCallback(func(estr string) {
		err = errors.New(estr)
	})

	pxs = make([]Proxy, 0)

	url := fmt.Sprintf("%sproxies/?api_key=%s", p.rootUrl, p.apiKey)
	for {
		resp, resperr := network.ParseHttpResult[PlProxiesResponse](p.logPrefix, "GetProxies", url, "GET", "", nil, nil, nil)
		if resperr != nil {
			err = resperr
			return
		} else {
			for _, pp := range resp.Result {
				px := pp.toProxy()
				if px.ExpireTime.Unix() > time.Now().Unix() {
					pxs = append(pxs, px)
				}
			}
		}

		url = resp.NextUrl
		if len(url) == 0 {
			break
		}
	}

	return
}

func (p *ProxyLineApi) NewProxies(location string, period int, quantity int) (pxs []Proxy, err error) {
	defer util.DefaultRecoverWithCallback(func(estr string) {
		err = errors.New(estr)
	})

	pxs = make([]Proxy, 0)

	req := PlNewOrderRequest{
		Type:      "dedicated",
		IPVersion: 4,
		Country:   location,
		Period:    period,
		Quantity:  quantity,
	}
	b, _ := json.Marshal(req)

	url := fmt.Sprintf("%snew-order/?api_key=%s", p.rootUrl, p.apiKey)
	resp, err := network.ParseHttpResult[PlNewOrderResponse](p.logPrefix, "Neworder", url, "POST", string(b), network.JsonHeaders(), nil, nil)
	for _, p := range *resp {
		pxs = append(pxs, p.toProxy())
	}
	return
}

func (p *ProxyLineApi) RenewProxies(ids []int, period int) (pxs []Proxy, err error) {
	defer util.DefaultRecoverWithCallback(func(estr string) {
		err = errors.New(estr)
	})

	pxs = make([]Proxy, 0)

	req := PlRenewRequest{
		Proxies: ids,
		Period:  period,
	}
	b, _ := json.Marshal(req)

	url := fmt.Sprintf("%srenew/?api_key=%s", p.rootUrl, p.apiKey)
	resp, err := network.ParseHttpResult[PlRenewResponse](p.logPrefix, "Renew", url, "POST", string(b), network.JsonHeaders(), nil, nil)
	for _, p := range *resp {
		pxs = append(pxs, p.toProxy())
	}
	return
}

func (p *ProxyLineApi) GetBalance() (bal float64, err error) {
	defer util.DefaultRecoverWithCallback(func(estr string) {
		bal = 0
		err = errors.New(estr)
	})

	url := fmt.Sprintf("%sbalance/?api_key=%s", p.rootUrl, p.apiKey)

	resp, err := network.ParseHttpResult[PlBalanceResponse](p.logPrefix, "GetProxies", url, "GET", "", nil, nil, nil)
	if err == nil {
		bal = resp.Balance
	} else {
		bal = 0
	}

	return
}
