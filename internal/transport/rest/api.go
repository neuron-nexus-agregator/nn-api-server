package rest

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"

	model "agregator/api/internal/model/db"
	"agregator/api/internal/service/db"
)

type API struct {
	db    *db.DB
	cache *cache.Cache
}

func New() *API {
	c := cache.New(10*time.Minute, 15*time.Minute)
	return &API{
		db:    db.New(),
		cache: c,
	}
}

func (a *API) Check(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
	})
}

func (a *API) GetMax(c *gin.Context) {
	max, err := a.db.GetLastIndex()
	if err != nil {
		c.JSON(500, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"max": max,
	})
}

func (a *API) Get(c *gin.Context) {
	date_str := c.DefaultQuery("date", "")
	limit_str := c.DefaultQuery("limit", "15")
	search_str := c.DefaultQuery("q", "")
	search_elements := strings.Split(search_str, ",")

	if len(search_elements) > 0 {
		for i, s := range search_elements {
			search_elements[i] = strings.TrimSpace(s)
		}
	}

	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	c.Header("Access-Control-Allow-Headers", "Content-Type")

	limit, err := strconv.ParseUint(limit_str, 10, 64)
	if err != nil {
		limit = 15
	}

	var date time.Time
	if date_str == "" {
		date = time.Now()
	} else {
		date, err = time.Parse(time.RFC3339, date_str)
		if err != nil {
			date = time.Now()
		}
	}
	if len(search_elements) == 0 {
		items, err := a.db.Get(date, limit)
		if err != nil {
			c.JSON(500, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(200, gin.H{"items": items})
	} else {
		items, err := a.db.Get(date, limit, search_elements...)
		if err != nil {
			c.JSON(500, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(200, gin.H{"items": items})
	}
}

func (a *API) GetTop(c *gin.Context) {
	limit_str := c.DefaultQuery("limit", "15")
	limit, err := strconv.ParseUint(limit_str, 10, 64)
	if err != nil {
		limit = 15
	}
	items, err := a.db.GetTopGroupsByFeedCount(limit)
	if err != nil {
		c.JSON(500, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{"items": items})
}

func (a *API) GetRT(c *gin.Context) {
	limit_str := c.DefaultQuery("limit", "15")
	limit, err := strconv.ParseUint(limit_str, 10, 64)
	if err != nil {
		limit = 15
	}
	is_rt_str := c.DefaultQuery("rt", "true")
	is_rt := strings.ToLower(is_rt_str) == "true"
	items, err := a.db.GetRTGroups(limit, is_rt)
	if err != nil {
		c.JSON(500, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{"items": items})
}

func (a *API) GetByID(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	c.Header("Access-Control-Allow-Headers", "Content-Type")
	id_str := c.Param("id")
	id, err := strconv.ParseUint(id_str, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{
			"error": err.Error(),
		})
		return
	}
	item, err := a.db.GetByID(id)
	if err != nil {
		c.JSON(500, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(200, item)
}

func (a *API) GetSimilar(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	c.Header("Access-Control-Allow-Headers", "Content-Type")
	id_str := c.Param("id")
	id, err := strconv.ParseUint(id_str, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{
			"error": err.Error(),
		})
		return
	}
	limit_str := c.DefaultQuery("limit", "10")
	limit, err := strconv.ParseUint(limit_str, 10, 64)
	if err != nil {
		limit = 10
	}

	if data, found := a.cache.Get(id_str); found {
		if values, ok := data.([]model.List); ok {
			if uint64(len(values)) >= limit {
				c.JSON(http.StatusOK, gin.H{"items": values[:limit]})
				return
			}
		}
	}

	items, err := a.db.GetSimilarGroups(id, limit)
	if err != nil {
		c.JSON(500, gin.H{
			"error": err.Error(),
		})
		return
	}
	a.cache.Add(id_str, items, cache.DefaultExpiration)
	c.JSON(200, gin.H{"items": items})
}
