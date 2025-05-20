package rest

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	model "agregator/api/internal/model/db"
	"agregator/api/internal/service/db"
	"agregator/api/internal/service/redis"
)

type API struct {
	db    *db.DB
	cache *redis.RedisCache
}

func New() *API {
	return &API{
		db:    db.New(),
		cache: redis.New(os.Getenv("REDIS_ADDR"), os.Getenv("REDIS_PASSWORD")),
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

	var items []model.List
	ok, err := a.cache.GetJSON("clusters:top", &items)
	if err == nil && ok {
		c.JSON(200, gin.H{"items": items})
		return
	} else if err != nil {
		log.Println(err)
	}

	items, err = a.db.GetTopGroupsByFeedCount(limit)
	if err != nil {
		c.JSON(500, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{"items": items})
	err = a.cache.Set("clusters:top", items, 10*time.Minute)
	if err != nil {
		log.Println(err)
	}
}

func (a *API) GetRT(c *gin.Context) {
	limit_str := c.DefaultQuery("limit", "15")
	limit, err := strconv.ParseUint(limit_str, 10, 64)
	if err != nil {
		limit = 15
	}
	is_rt_str := c.DefaultQuery("rt", "true")
	is_rt := strings.ToLower(is_rt_str) == "true"

	var items []model.List
	if is_rt {
		ok, err := a.cache.GetJSON("clusters:rt", &items)
		if err == nil && ok {
			c.JSON(200, gin.H{"items": items})
			return
		} else if err != nil {
			log.Println(err)
		}
	} else {
		ok, err := a.cache.GetJSON("clusters:not_rt", &items)
		if err == nil && ok {
			c.JSON(200, gin.H{"items": items})
			return
		} else if err != nil {
			log.Println(err)
		}
	}

	items, err = a.db.GetRTGroups(limit, is_rt)
	if err != nil {
		c.JSON(500, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{"items": items})
	if is_rt {
		err = a.cache.Set("clusters:rt", items, 10*time.Minute)
	} else {
		err = a.cache.Set("clusters:not_rt", items, 10*time.Minute)
	}
	if err != nil {
		log.Println(err)
	}
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

	var item model.News
	ok, err := a.cache.GetJSON("clusters:"+id_str, &item)
	if err == nil && ok {
		c.JSON(200, item)
		return
	} else if err != nil {
		log.Println(err)
	}

	item, err = a.db.GetByID(id)
	if err != nil {
		c.JSON(500, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(200, item)
	err = a.cache.Set("clusters:"+id_str, item, 1*time.Hour)
	if err != nil {
		log.Println(err)
	}
	go func() {
		err := a.db.IncrementVies(item.ID)
		if err != nil {
			log.Default().Println(err)
		}
	}()
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
	var items []model.List
	ok, err := a.cache.GetJSON("clusters:similar:"+id_str, &items)
	if err == nil && ok {
		c.JSON(200, gin.H{"items": items})
		return
	} else if err != nil {
		log.Println(err)
	}

	items, err = a.db.GetSimilarGroups(id, limit)
	if err != nil {
		c.JSON(500, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{"items": items})
	err = a.cache.Set("clusters:similar:"+id_str, items, 1*time.Hour)
	if err != nil {
		log.Println(err)
	}
}
