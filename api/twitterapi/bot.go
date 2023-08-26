/*
 * @Author: aztec
 * @Date: 2022-11-23 09:42:33
 * @Description: 这个其实并不属于api，而是网页爬虫
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package twitterapi

import (
	"context"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"aztecqt/dagger/util"
	"aztecqt/dagger/util/logger"
	"aztecqt/dagger/util/mathtools"
	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
)

type Bot struct {
	logPrefix          string
	username, password string
	dedup              *mathtools.Deduplicator
	onNewTweet         FuncNewTweet
	firstTime          bool
}

func (b *Bot) Test() {
	if data, err := ioutil.ReadFile("1.html"); err == nil {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(data)))
		if err != nil {
			logger.LogImportant(b.logPrefix, "error while create goQuery document, err=%s", err.Error())
		} else {
			tweetFromSelection(doc.Selection)
		}
	}
}

type FuncNewTweet func(t *Tweet)

func (b *Bot) Run(headless bool, username, password string, onNewTweet FuncNewTweet) {
	b.logPrefix = "twitter_bot"
	b.username = username
	b.password = password
	b.dedup = mathtools.NewDeduplicator(1000)
	b.onNewTweet = onNewTweet
	b.firstTime = true
	options := []chromedp.ExecAllocatorOption{
		chromedp.Flag("headless", headless),
		chromedp.Flag("blink-settings", "imageEnable=false"),
		chromedp.Flag("no-default-browser-check", true),
		chromedp.Flag("disk-cache-size", "67108864"), // 最大64M缓存
		chromedp.Flag("lang", "en-US"),
		chromedp.UserAgent(`Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/105.0.0.0 Safari/537.36`),
	}

	ctx, _ := chromedp.NewExecAllocator(context.Background(), options...)
	ctx, cancel := chromedp.NewContext(ctx, chromedp.WithLogf(log.Printf))
	defer cancel()

	// 先打开首页
	b.openMainPage(ctx)

	// 主循环
	for {
		// 确保登录成功
		b.makeSureLogin(ctx)

		// 主页操作逻辑
		b.mainPageOperation(ctx)

		time.Sleep(time.Second * 3)
	}
}

// 打开主页
func (b *Bot) openMainPage(ctx context.Context) {
	logger.LogInfo(b.logPrefix, "opening twitter home page")
	chromedp.Run(ctx,
		util.LoadCookies(),
		chromedp.Navigate("https://twitter.com/home"))
	logger.LogInfo(b.logPrefix, "home page opened")
}

// 确保登录
func (b *Bot) makeSureLogin(ctx context.Context) {
	loginPerformed := false
	for i := 0; i < 60; i++ {
		// 判断当前状态
		status := "invalid"
		func() {
			body := ""
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			err := chromedp.Run(ctx,
				chromedp.OuterHTML("body", &body, chromedp.ByQuery))
			if err == nil {
				if strings.Contains(body, "cellInnerDiv") {
					status = "main_page"
				} else if strings.Contains(body, "Sign in to Twitter") {
					status = "login"
				}
			} else {
				logger.LogImportant(b.logPrefix, err.Error())
			}
		}()

		if status == "main_page" {
			// 主页显示出来了
			if loginPerformed {
				util.SaveCookies()
			}
			break
		} else if status == "login" {
			// 手动登录
			logger.LogInfo(b.logPrefix, "perform manual login")
			func() {
				ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
				defer cancel()
				err := chromedp.Run(ctx,
					chromedp.DoubleClick("input[autocomplete=username]"),
					chromedp.SendKeys("input[autocomplete=username]", b.username),
					chromedp.Sleep(time.Second),
					chromedp.SendKeys("input[autocomplete=username]", kb.Enter),
					chromedp.WaitReady("input[autocomplete=current-password]"),
					chromedp.Sleep(time.Second),
					chromedp.SendKeys("input[autocomplete=current-password]", b.password),
					chromedp.Sleep(time.Second),
					chromedp.SendKeys("input[autocomplete=current-password]", kb.Enter),
					chromedp.WaitReady("div[data-testid=cellInnerDiv]"),
					util.SaveCookies(),
				)

				if err == nil {
					logger.LogInfo(b.logPrefix, "manual login successed")
					loginPerformed = true
				} else {
					logger.LogInfo(b.logPrefix, "manual login failed with error: %s", err.Error())
					b.openMainPage(ctx)
				}
			}()
		}

		time.Sleep(time.Second)
	}
}

func (b *Bot) mainPageOperation(ctx context.Context) {
	backButtonId := cdp.NodeID(0)
	modeButtonId := cdp.NodeID(0)

	// 返回按钮
	ctx0, cancel0 := context.WithTimeout(ctx, time.Second)
	nodes := make([]*cdp.Node, 0)
	chromedp.Run(ctx0, chromedp.Nodes("div[role=button][data-testid=app-bar-back]", &nodes))
	cancel0()
	if len(nodes) > 0 {
		backButtonId = nodes[0].NodeID
	}

	// 小星星按钮
	ctx1, cancel1 := context.WithTimeout(ctx, time.Second)
	nodes = make([]*cdp.Node, 0)
	latestTweetsMode := false
	chromedp.Run(ctx1, chromedp.Nodes("[aria-label~=Tweets][aria-label~=Top]", &nodes))
	cancel1()
	if len(nodes) > 0 {
		modeButtonId = nodes[0].NodeID
		if attr, ok := nodes[0].Attribute("aria-label"); ok {
			latestTweetsMode = attr == "Top Tweets off"
		}
	}

	// 显示新推文按钮
	newTweetsBtnId := util.FindButtonWithText(ctx, "^Show [0-9]+ Tweet")

	// 固定按一下esc
	func() {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		chromedp.Run(ctx, chromedp.KeyEvent(kb.Escape))
	}()

	if backButtonId > 0 {
		// 有返回按钮，优先点击
		func() {
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			if err := chromedp.Run(ctx, chromedp.Click([]cdp.NodeID{backButtonId}, chromedp.ByNodeID)); err == nil {
				logger.LogInfo(b.logPrefix, "click back button")
			} else {
				logger.LogImportant(b.logPrefix, err.Error())
			}
		}()
	} else {
		if latestTweetsMode {
			// 收集一下推文
			b.processTweets(ctx)

			if newTweetsBtnId > 0 {
				// 有新推文，点一下按钮
				func() {
					ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
					defer cancel()
					if err := chromedp.Run(ctx, chromedp.Click([]cdp.NodeID{newTweetsBtnId}, chromedp.ByNodeID)); err == nil {
						logger.LogInfo(b.logPrefix, "click [show xxx tweets] button")
					} else {
						logger.LogImportant(b.logPrefix, err.Error())
					}
				}()
			} else {
				// 点击home按钮
				func() {
					ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
					defer cancel()
					if err := chromedp.Run(ctx, chromedp.Click("a[href*=home][aria-label*=Home]")); err == nil {
						logger.LogInfo(b.logPrefix, "click home button")
					} else {
						logger.LogImportant(b.logPrefix, err.Error())
					}
				}()

				/*// 刷新页面
				func() {
					ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
					defer cancel()
					if err := chromedp.Run(ctx, chromedp.Reload()); err == nil {
						logger.LogInfo(b.logPrefix, "refresh main page")
					} else {
						logger.LogImportant(b.logPrefix, err.Error())
					}
				}()*/

				/*// 查看第一条推文
				func() {
					ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
					defer cancel()
					if err := chromedp.Run(ctx, chromedp.Click("time[datetime]")); err == nil {
						logger.LogInfo(b.logPrefix, "click first article")
					} else {
						logger.LogImportant(b.logPrefix, err.Error())
					}
				}()*/

				/*// 上下翻滚几次
				func() {
					ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
					defer cancel()
					if err := chromedp.Run(ctx,
						chromedp.KeyEvent(kb.PageDown),
						chromedp.Sleep(time.Second),
						chromedp.KeyEvent(kb.PageUp),
						chromedp.Sleep(time.Second/2),
						chromedp.KeyEvent(kb.PageUp)); err == nil {
						logger.LogInfo(b.logPrefix, "click first article")
					} else {
						logger.LogImportant(b.logPrefix, err.Error())
					}
				}()*/
			}
		} else {
			// 切换模式
			func() {
				ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
				defer cancel()
				err := chromedp.Run(ctx,
					chromedp.Click([]cdp.NodeID{modeButtonId}, chromedp.ByNodeID), // 小星星按钮
					chromedp.WaitReady("[role=menuitem]"),                         // 第一个menuitem就是切换模式的按钮
					chromedp.Sleep(time.Second),
					chromedp.Click("[role=menuitem]"))
				if err == nil {
					logger.LogInfo(b.logPrefix, "switched to latest tweet mode")
				} else {
					logger.LogImportant(b.logPrefix, err.Error())
				}
			}()
		}
	}
}

// 处理主页推文
func (b *Bot) processTweets(ctx context.Context) {
	// 解析目前的主页内容
	hp := b.parseHomePage(ctx)

	// 找出新出现的推文，向外输出
	if hp != nil && len(hp.Tweets) > 0 {
		for i := range hp.Tweets {
			t := hp.Tweets[len(hp.Tweets)-i-1]
			if !b.dedup.IsDuplicated(t.TweetId) {
				logger.LogInfo(b.logPrefix, "new tweet:\n%s", t.String())
				if b.onNewTweet != nil && !b.firstTime {
					b.onNewTweet(t)
				}
			}
		}
		b.firstTime = false
	}
}

// 获取当前页面的全部推文
func (b *Bot) parseHomePage(ctx context.Context) *HomePage {
	body := ""
	ctx0, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	logger.LogInfo(b.logPrefix, "query main page...")
	err := chromedp.Run(ctx0, chromedp.OuterHTML("body", &body, chromedp.ByQuery))

	if err != nil {
		logger.LogImportant(b.logPrefix, "error while query main page: %s", err.Error())
		return nil
	}

	// 对body使用goquery进行分析
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		logger.LogImportant(b.logPrefix, "error while create goQuery document, err=%s", err.Error())
		return nil
	}

	hp := new(HomePage)
	hp.Tweets = make([]*Tweet, 0)

	// 找博文数据
	selector := "article"
	s1 := doc.Find(selector)
	s1.Each(func(i int, s *goquery.Selection) {
		twt := tweetFromSelection(s)
		if twt != nil {
			if twt.Time.Unix() == 0 {
				logger.LogImportant(b.logPrefix, "invalid tweet time. tweet:\n%s\n", twt.String())
			} else if len(twt.Author) == 0 {
				logger.LogImportant(b.logPrefix, "invalid tweet author. tweet:\n%s\n", twt.String())
			} else if twt.TweetId == 0 {
				logger.LogImportant(b.logPrefix, "invalid tweet id. tweet:\n%s\n", twt.String())
			} else if len(twt.Url) == 0 {
				logger.LogImportant(b.logPrefix, "invalid tweet url. tweet:\n%s\n", twt.String())
			} else {
				hp.Tweets = append(hp.Tweets, twt)
			}
		}
	})

	logger.LogInfo(b.logPrefix, "get %d tweets in main page", len(hp.Tweets))
	return hp
}
