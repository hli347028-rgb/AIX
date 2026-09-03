package service

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"backend/internal/biz"
	"backend/internal/data"

	"github.com/go-kratos/kratos/v2/errors"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/shopspring/decimal"
)

var klineIntervalMinutes = map[string]int{
	"15m": 15,
	"1h":  60,
	"4h":  240,
	"1d":  1440,
}

type klinePoint struct {
	Time   int64   `json:"time"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume float64 `json:"volume"`
}

// HandlePublicMarketKline 按系统内 AIX、WIN 现价绘制 AIX/WIN K 线（不依赖 AVE）。
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

	limit := 120
	if raw := strings.TrimSpace(q.Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			if n > 500 {
				n = 500
			}
			limit = n
		}
	}

	pair := strings.TrimSpace(q.Get("pair"))
	if pair == "" {
		pair = "AIX-WIN"
	}

	points, spot, err := s.buildAixWinKlines(ctx, minutes, limit)
	if err != nil {
		return err
	}
	if len(points) == 0 {
		return errors.ServiceUnavailable("PRICE_UNAVAILABLE", "aix/win price unavailable")
	}

	return ctx.JSON(http.StatusOK, map[string]any{
		"pair":     pair,
		"interval": interval,
		"source":   "aix_win",
		"spot":     spot,
		"points":   points,
		"candles":  points,
	})
}

func (s *AdminLegacyService) buildAixWinKlines(ctx context.Context, minutes, limit int) ([]klinePoint, float64, error) {
	winPrice := biz.GetWinPrice()
	if winPrice <= 0 {
		if s.walletRepo != nil {
			if raw, err := s.walletRepo.GetCurrentWinPrice(ctx); err == nil {
				if p, perr := decimal.NewFromString(strings.TrimSpace(raw)); perr == nil && p.IsPositive() {
					winPrice, _ = p.Float64()
				}
			}
		}
	}
	if winPrice <= 0 {
		return nil, 0, errors.ServiceUnavailable("WIN_PRICE_UNAVAILABLE", "win price not configured")
	}

	aixByDay, latestAix, err := s.loadAixPriceSeries(ctx, limit+7)
	if err != nil {
		return nil, 0, err
	}
	if latestAix <= 0 {
		latestAix = biz.AixPriceInitial
		if latestAix <= 0 {
			latestAix = 1
		}
	}

	spot := latestAix / winPrice
	if !isFinitePositive(spot) {
		return nil, 0, errors.ServiceUnavailable("PRICE_UNAVAILABLE", "invalid aix/win spot")
	}

	step := time.Duration(minutes) * time.Minute
	now := time.Now()
	end := now.Truncate(step)
	if end.After(now) {
		end = end.Add(-step)
	}

	out := make([]klinePoint, 0, limit)
	prevClose := 0.0
	for i := limit - 1; i >= 0; i-- {
		t := end.Add(-time.Duration(i) * step)
		dayKey := t.Format("2006-01-02")
		aix := lookupAixPrice(aixByDay, dayKey, latestAix)
		base := aix / winPrice
		if !isFinitePositive(base) {
			base = spot
		}

		open := base
		if prevClose > 0 {
			open = prevClose
		}
		// 日内微波动：由时间戳确定性扰动，刷新时图形稳定，末根贴近现价
		wiggle := priceWiggle(t.Unix(), minutes) * base
		closePx := base + wiggle
		if i == 0 {
			closePx = spot
		}
		high := math.Max(open, closePx) + math.Abs(wiggle)*0.55
		low := math.Min(open, closePx) - math.Abs(wiggle)*0.4
		if low <= 0 {
			low = math.Min(open, closePx) * 0.998
		}
		vol := 80000 + math.Abs(math.Sin(float64(t.Unix())/3600))*420000 + float64(limit-i)*1800

		out = append(out, klinePoint{
			Time:   t.Unix(),
			Open:   roundPrice(open),
			High:   roundPrice(high),
			Low:    roundPrice(low),
			Close:  roundPrice(closePx),
			Volume: math.Round(vol),
		})
		prevClose = closePx
	}
	return out, roundPrice(spot), nil
}

func (s *AdminLegacyService) loadAixPriceSeries(ctx context.Context, want int) (map[string]float64, float64, error) {
	out := map[string]float64{}
	latest := 0.0
	if s.data == nil || s.data.DB() == nil {
		return out, latest, nil
	}
	if want < 30 {
		want = 30
	}
	var rows []data.AixPricePO
	if err := s.data.DB().WithContext(ctx).
		Order("effective_date desc").
		Limit(want).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	for _, row := range rows {
		f, _ := row.Price.Float64()
		if f <= 0 {
			continue
		}
		day := strings.TrimSpace(row.EffectiveDate)
		if len(day) >= 10 {
			day = day[:10]
		}
		out[day] = f
		if latest <= 0 {
			latest = f // rows are effective_date desc
		}
	}
	if latest <= 0 && s.walletRepo != nil {
		today := time.Now().Format("2006-01-02")
		if raw, err := s.walletRepo.GetAixPrice(ctx, today); err == nil && strings.TrimSpace(raw) != "" {
			if p, err := decimal.NewFromString(strings.TrimSpace(raw)); err == nil && p.IsPositive() {
				latest, _ = p.Float64()
			}
		}
		if latest <= 0 {
			if raw, err := s.walletRepo.GetLatestAixPriceBefore(ctx, today); err == nil && strings.TrimSpace(raw) != "" {
				if p, err := decimal.NewFromString(strings.TrimSpace(raw)); err == nil && p.IsPositive() {
					latest, _ = p.Float64()
				}
			}
		}
	}
	return out, latest, nil
}

func lookupAixPrice(byDay map[string]float64, day string, fallback float64) float64 {
	if v, ok := byDay[day]; ok && v > 0 {
		return v
	}
	// 向前找最近有价日
	t, err := time.ParseInLocation("2006-01-02", day, time.Local)
	if err != nil {
		return fallback
	}
	for i := 0; i < 60; i++ {
		key := t.AddDate(0, 0, -i).Format("2006-01-02")
		if v, ok := byDay[key]; ok && v > 0 {
			return v
		}
	}
	return fallback
}

func priceWiggle(unix int64, minutes int) float64 {
	x := float64(unix/int64(minutes*60) + int64(minutes))
	return math.Sin(x*0.37)*0.0048 + math.Cos(x*0.19)*0.0026
}

func roundPrice(v float64) float64 {
	if !isFinitePositive(v) && v != 0 {
		return 0
	}
	return math.Round(v*1e8) / 1e8
}

func isFinitePositive(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && v > 0
}
