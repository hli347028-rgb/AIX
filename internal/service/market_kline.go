package service

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

var klineIntervalMinutes = map[string]int{
	"15m": 15,
	"1h":  60,
	"4h":  240,
	"1d":  1440,
}

// HandlePublicMarketKline proxies AVE token klines for the homepage chart.
// GET /v1/market/kline?interval=1h&limit=300&pair=AIX-WIN
func (s *AdminLegacyService) HandlePublicMarketKline(ctx khttp.Context) error {
	q := ctx.Request().URL.Query()
	interval := strings.TrimSpace(q.Get("interval"))
	if interval == "" {
		interval = "1h"
	}
	minutes, ok := klineIntervalMinutes[interval]
	if !ok {
		return errors.BadRequest("INVALID_INTERVAL", "interval must be 15m|1h|4h|1d")
	}

	limit := 300
	if raw := strings.TrimSpace(q.Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			if n > 1000 {
				n = 1000
			}
			limit = n
		}
	}

	pair := strings.TrimSpace(q.Get("pair"))
	if pair == "" {
		pair = "AIX-WIN"
	}

	apiKey := s.walletCfg.GetAveAPIKey()
	if apiKey == "" {
		return errors.ServiceUnavailable("AVE_NOT_CONFIGURED", "AVE API key not configured")
	}

	tokenID := s.walletCfg.GetAveKlineTokenID()
	baseURL := strings.TrimRight(s.walletCfg.GetAveKlineBaseURL(), "/")
	endpoint := fmt.Sprintf("%s/v2/klines/token/%s?interval=%d&limit=%d",
		baseURL, url.PathEscape(tokenID), minutes, limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-API-KEY", apiKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return errors.ServiceUnavailable("AVE_UPSTREAM", "failed to reach AVE API")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.ServiceUnavailable("AVE_ERROR", fmt.Sprintf("AVE returned %d", resp.StatusCode))
	}

	var aveResp struct {
		Status int    `json:"status"`
		Msg    string `json:"msg"`
		Data   struct {
			Points []struct {
				Time   int64  `json:"time"`
				Open   string `json:"open"`
				High   string `json:"high"`
				Low    string `json:"low"`
				Close  string `json:"close"`
				Volume string `json:"volume"`
			} `json:"points"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &aveResp); err != nil {
		return errors.ServiceUnavailable("AVE_PARSE", "invalid AVE response")
	}
	if aveResp.Status != 1 || len(aveResp.Data.Points) == 0 {
		msg := strings.TrimSpace(aveResp.Msg)
		if msg == "" {
			msg = "empty kline data"
		}
		return errors.ServiceUnavailable("AVE_EMPTY", msg)
	}

	candles := make([]map[string]interface{}, 0, len(aveResp.Data.Points))
	for _, point := range aveResp.Data.Points {
		open, okOpen := parseAveDecimal(point.Open)
		high, okHigh := parseAveDecimal(point.High)
		low, okLow := parseAveDecimal(point.Low)
		closePrice, okClose := parseAveDecimal(point.Close)
		volume, _ := parseAveDecimal(point.Volume)
		if point.Time <= 0 || !okOpen || !okHigh || !okLow || !okClose {
			continue
		}
		candles = append(candles, map[string]interface{}{
			"time":   point.Time,
			"open":   open,
			"high":   high,
			"low":    low,
			"close":  closePrice,
			"volume": volume,
		})
	}
	if len(candles) == 0 {
		return errors.ServiceUnavailable("AVE_EMPTY", "no valid kline points")
	}

	return ctx.Result(200, map[string]interface{}{
		"pair":     pair,
		"interval": interval,
		"candles":  candles,
		"source":   "ave",
	})
}

func parseAveDecimal(raw string) (float64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, false
	}
	return value, true
}
