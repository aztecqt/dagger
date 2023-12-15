/*
 * @Author: aztec
 * @Date: 2023-05-23
 * @Description:
 */

package proxyprovider

import (
	"strings"
	"time"
)

type PlProxy struct {
	ID         int    `json:"id"`
	IP         string `json:"ip"`
	Port       int    `json:"port_socks5"`
	UserName   string `json:"username"`
	Password   string `json:"password"`
	Country    string `json:"country"`
	DateStr    string `json:"date"`
	DateEndStr string `json:"date_end"`
	Date       time.Time
	DateEnd    time.Time
}

func (p *PlProxy) parse() {
	p.DateStr = strings.Replace(p.DateStr, " ", "T", 1)
	if date, err := time.Parse(time.RFC3339, p.DateStr); err == nil {
		p.Date = date
	}

	if date, err := time.Parse(time.RFC3339, p.DateEndStr); err == nil {
		p.DateEnd = date
	}
}

func (p *PlProxy) toProxy() Proxy {
	p.parse()
	proxy := Proxy{}
	proxy.Provider = "proxyline"
	proxy.OrderID = p.ID
	proxy.IP = p.IP
	proxy.Port = p.Port
	proxy.UserName = p.UserName
	proxy.Password = p.Password
	proxy.Location = p.Country
	proxy.CreateTime = p.Date
	proxy.ExpireTime = p.DateEnd
	return proxy
}

type PlProxiesResponse struct {
	Count   int        `json:"count"`
	NextUrl string     `json:"next"`
	PrevUrl string     `json:"previous"`
	Result  []*PlProxy `json:"results"`
}

type PlRenewRequest struct {
	Proxies []int `json:"proxies"`
	Period  int   `json:"period"`
}

type PlRenewResponse []*PlProxy

type PlNewOrderRequest struct {
	Type      string `json:"type"`
	IPVersion int    `json:"ip_version"`
	Country   string `json:"country"`
	Period    int    `json:"period"`
	Quantity  int    `json:"quantity"`
}

type PlNewOrderResponse []*PlProxy

type PlBalanceResponse struct {
	Balance float64 `json:"balance"`
}
