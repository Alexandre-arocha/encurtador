package server

import (
	"errors"
	"net/http"
	"time"

	"encurtador/internal/links"
	"encurtador/internal/stats"

	"github.com/gin-gonic/gin"
)

type statsResponse struct {
	Range            string                  `json:"range"`
	StartDay         string                  `json:"start_day"`
	EndDay           string                  `json:"end_day"`
	TotalClicks      int64                   `json:"total_clicks"`
	Daily            []dailyPointResponse    `json:"daily"`
	TopReferrers     []referrerPointResponse `json:"top_referrers"`
	Devices          []breakdownResponse     `json:"devices"`
	Countries        []breakdownResponse     `json:"countries"`
	Browsers         []breakdownResponse     `json:"browsers"`
	OperatingSystems []breakdownResponse     `json:"operating_systems"`
	Cities           []breakdownResponse     `json:"cities"`
}

type dailyPointResponse struct {
	Day    string `json:"day"`
	Clicks int64  `json:"clicks"`
}

type referrerPointResponse struct {
	Referrer *string `json:"referrer"`
	Clicks   int64   `json:"clicks"`
}

type breakdownResponse struct {
	Key    string `json:"key"`
	Clicks int64  `json:"clicks"`
}

func (s *Server) handleGetLinkStats(c *gin.Context) {
	id, ok := s.parseLinkID(c)
	if !ok {
		return
	}

	if _, err := s.links.GetByID(c.Request.Context(), id); err != nil {
		s.respondLinkError(c, err)
		return
	}

	rangeValue, err := stats.ParseRange(c.Query("range"))
	if err != nil {
		s.respondError(c, http.StatusBadRequest, "invalid_range", "range inválido (use 7d, 30d ou all)")
		return
	}
	if s.stats == nil {
		s.respondError(c, http.StatusInternalServerError, "config_error", "stats não configurado")
		return
	}

	result, err := s.stats.Get(c.Request.Context(), id, rangeValue, time.Now(), stats.DefaultTopN)
	if err != nil {
		if errors.Is(err, links.ErrNotFound) {
			s.respondLinkError(c, err)
			return
		}
		s.logger.Error("erro ao buscar stats", "link_id", id.String(), "err", err)
		s.respondError(c, http.StatusInternalServerError, "internal_error", "erro ao buscar estatísticas")
		return
	}

	c.JSON(http.StatusOK, toStatsResponse(result))
}

func toStatsResponse(result stats.Result) statsResponse {
	resp := statsResponse{
		Range:            string(result.Range),
		StartDay:         result.StartDay.Format(time.DateOnly),
		EndDay:           result.EndDay.Format(time.DateOnly),
		TotalClicks:      result.TotalClicks,
		Daily:            make([]dailyPointResponse, 0, len(result.Daily)),
		TopReferrers:     make([]referrerPointResponse, 0, len(result.TopReferrers)),
		Devices:          make([]breakdownResponse, 0, len(result.Devices)),
		Countries:        make([]breakdownResponse, 0, len(result.Countries)),
		Browsers:         make([]breakdownResponse, 0, len(result.Browsers)),
		OperatingSystems: make([]breakdownResponse, 0, len(result.OperatingSystems)),
		Cities:           make([]breakdownResponse, 0, len(result.Cities)),
	}
	for _, point := range result.Daily {
		resp.Daily = append(resp.Daily, dailyPointResponse{
			Day:    point.Day.Format(time.DateOnly),
			Clicks: point.Clicks,
		})
	}
	for _, point := range result.TopReferrers {
		resp.TopReferrers = append(resp.TopReferrers, referrerPointResponse{
			Referrer: point.Referrer,
			Clicks:   point.Clicks,
		})
	}
	for _, point := range result.Devices {
		resp.Devices = append(resp.Devices, breakdownResponse(point))
	}
	for _, point := range result.Countries {
		resp.Countries = append(resp.Countries, breakdownResponse(point))
	}
	for _, point := range result.Browsers {
		resp.Browsers = append(resp.Browsers, breakdownResponse(point))
	}
	for _, point := range result.OperatingSystems {
		resp.OperatingSystems = append(resp.OperatingSystems, breakdownResponse(point))
	}
	for _, point := range result.Cities {
		resp.Cities = append(resp.Cities, breakdownResponse(point))
	}
	return resp
}
