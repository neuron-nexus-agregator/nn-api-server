package db

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	model "agregator/api/internal/model/db"
)

type DB struct {
	host     string
	port     string
	login    string
	password string
	db_name  string
}

func New() *DB {
	return &DB{
		host:     os.Getenv("DB_HOST"),
		port:     os.Getenv("DB_PORT"),
		login:    os.Getenv("DB_LOGIN"),
		password: os.Getenv("DB_PASSWORD"),
		db_name:  "newagregator",
	}
}

func (g *DB) connectToDB() (*sqlx.DB, error) {
	connectionData := fmt.Sprintf("user=%s dbname=%s sslmode=disable password=%s host=%s port=%s", g.login, g.db_name, g.password, g.host, g.port)
	return sqlx.Connect("postgres", connectionData)
}

func (g *DB) GetLastIndex() (uint64, error) {
	db, err := g.connectToDB()
	if err != nil {
		return 0, err
	}
	defer db.Close()
	var index uint64
	err = db.QueryRow("SELECT MAX(id) FROM groups").Scan(&index)
	if err != nil {
		return 0, err
	}
	return index, nil
}

func (g *DB) Get(lastDate time.Time, limit uint64, search ...string) ([]model.List, error) {
	db, err := g.connectToDB()
	if err != nil {
		log.Default().Println("Error connecting to DB: ", err.Error())
		return nil, err
	}
	defer db.Close()

	// Базовый SQL-запрос
	baseReq := `
        SELECT 
            groups.id, 
            groups.time, 
            feed.title, 
            feed.description, 
			feed.source_name,
            groups.is_rt,
            (
                SELECT feed.enclosure
                FROM compares
                JOIN feed ON feed.id = compares.feed_id
                WHERE compares.group_id = groups.id 
                  AND feed.enclosure IS NOT NULL 
                  AND feed.enclosure != ''
                LIMIT 1
            ) AS enclosure
        FROM groups
        JOIN feed ON groups.feed_id = feed.id
        WHERE groups.time < $1
    `

	// Если есть поисковые запросы, добавляем фильтры
	var whereClauses []string
	var args []interface{}
	args = append(args, lastDate)

	if len(search) > 0 && search[0] != "" {
		for _, q := range search {
			q = strings.ReplaceAll(q, " ", "%")
			likePattern := `%` + q + `%`
			whereClauses = append(whereClauses, `feed.title ILIKE $`+strconv.Itoa(len(args)+1))
			whereClauses = append(whereClauses, `feed.description ILIKE $`+strconv.Itoa(len(args)+2))
			whereClauses = append(whereClauses, `feed.full_text ILIKE $`+strconv.Itoa(len(args)+3))
			args = append(args, likePattern, likePattern, likePattern)
		}
		baseReq += ` AND (` + strings.Join(whereClauses, ` OR `) + `)`
	}

	// Добавляем сортировку и лимит
	baseReq += `
        ORDER BY groups.time DESC
        LIMIT $` + strconv.Itoa(len(args)+1)
	args = append(args, limit)

	// Выполняем запрос
	stmt, err := db.Preparex(baseReq)
	if err != nil {
		log.Default().Println("Error preparing query: ", err.Error())
		return nil, err
	}
	defer stmt.Close()

	var groups []model.List
	err = stmt.Select(&groups, args...)
	if err != nil {
		log.Default().Println("Error getting groups: ", err.Error())
		return nil, err
	}

	return groups, nil
}

func (g *DB) GetTopGroupsByFeedCount(limit uint64) ([]model.List, error) {
	db, err := g.connectToDB()
	if err != nil {
		log.Default().Println("Error connecting to DB: ", err.Error())
		return nil, err
	}
	defer db.Close()

	req := `
        SELECT 
            groups.id, 
            groups.time, 
            feed.title, 
            feed.description, 
			feed.source_name,
            groups.is_rt, 
            (
                SELECT feed.enclosure
                FROM compares
                JOIN feed ON feed.id = compares.feed_id
                WHERE compares.group_id = groups.id 
                  AND feed.enclosure IS NOT NULL 
                  AND feed.enclosure != ''
                LIMIT 1
            ) AS enclosure
        FROM groups
        JOIN feed ON groups.feed_id = feed.id
        WHERE groups.time >= NOW() - INTERVAL '27 HOURS'
        GROUP BY groups.id, feed.title, feed.description, groups.time, groups.is_rt, feed.source_name, enclosure
        ORDER BY (
            SELECT COUNT(*)
            FROM compares
            WHERE compares.group_id = groups.id
        ) DESC,
		groups.time DESC
        LIMIT $1
    `

	stmt, err := db.Preparex(req)
	if err != nil {
		log.Default().Println("Error preparing query: ", err.Error())
		return nil, err
	}
	defer stmt.Close()

	var groups []model.List
	err = stmt.Select(&groups, limit)
	if err != nil {
		log.Default().Println("Error getting top groups by feed count: ", err.Error())
		return nil, err
	}

	return groups, nil
}

func (g *DB) GetRTGroups(limit uint64, is_rt bool) ([]model.List, error) {
	db, err := g.connectToDB()
	if err != nil {
		log.Default().Println("Error connecting to DB: ", err.Error())
		return nil, err
	}
	defer db.Close()

	req := `
        SELECT 
            groups.id, 
            groups.time, 
            feed.title, 
            feed.description, 
			feed.source_name,
            groups.is_rt,
            (
                SELECT COALESCE(feed.enclosure, '')
                FROM compares
                JOIN feed ON feed.id = compares.feed_id
                WHERE compares.group_id = groups.id 
                  AND feed.enclosure IS NOT NULL 
                  AND feed.enclosure != ''
                LIMIT 1
            ) AS enclosure
        FROM groups
        JOIN feed ON groups.feed_id = feed.id
        WHERE groups.is_rt = $1
        ORDER BY groups.time DESC
        LIMIT $2
    `

	stmt, err := db.Preparex(req)
	if err != nil {
		log.Default().Println("Error preparing query: ", err.Error())
		return nil, err
	}
	defer stmt.Close()

	var groups []model.List
	err = stmt.Select(&groups, is_rt, limit)
	if err != nil {
		log.Default().Println("Error getting RT groups: ", err.Error())
		return nil, err
	}

	return groups, nil
}

func (g *DB) GetSimilarGroups(id, limit uint64) ([]model.List, error) {
	db, err := g.connectToDB()
	if err != nil {
		log.Default().Println("Error connecting to DB: ", err.Error())
		return nil, err
	}
	defer db.Close()
	req := `SELECT
            g.id,
            g.time,
            g.is_rt,
            feed.title,
            feed.description,
			feed.source_name,
            (
                SELECT COALESCE(feed.enclosure, '')
                FROM compares
                JOIN feed ON feed.id = compares.feed_id
                WHERE compares.group_id = g.id 
                  AND feed.enclosure IS NOT NULL 
                  AND feed.enclosure != ''
                LIMIT 1
            ) AS enclosure
        FROM
            groups g
        JOIN
            feed ON feed.id = g.feed_id
        WHERE
            g.id <> $1
        ORDER BY
            1 - (g.embedding <=> (SELECT embedding FROM groups WHERE id = $1)) DESC,
            g.time DESC
        LIMIT $2`

	var groups []model.List
	err = db.Select(&groups, req, id, limit)
	if err != nil {
		log.Default().Println("Error getting similar groups: ", err.Error())
		return nil, err
	}

	return groups, nil
}

func (g *DB) GetByID(id uint64) (model.News, error) {
	req := `SELECT groups.id, feed.title, feed.description, feed.full_text, groups.time,
	(
                SELECT COALESCE(feed.enclosure, '')
                FROM compares
                JOIN feed ON feed.id = compares.feed_id
                WHERE compares.group_id = groups.id 
                  AND feed.enclosure IS NOT NULL 
                  AND feed.enclosure != ''
                LIMIT 1
            ) AS enclosure
	FROM groups
	JOIN feed ON feed.id = groups.feed_id
	WHERE groups.id = $1`

	db, err := g.connectToDB()
	if err != nil {
		log.Default().Println("Error connecting to DB: ", err.Error())
		return model.News{}, err
	}
	defer db.Close()
	var group model.News
	err = db.Get(&group, req, id)
	if err != nil {
		log.Default().Println("Error gettin news: ", err.Error())
		return model.News{}, err
	}

	var sources []model.Source
	req = `SELECT feed.title, feed.link, feed.source_name, feed.time, feed.description, feed.full_text, feed.enclosure
	FROM compares
	JOIN feed ON feed.id = compares.feed_id
	WHERE compares.group_id = $1
	ORDER BY feed.time desc`
	err = db.Select(&sources, req, id)
	if err == nil {
		group.Sources = sources
	} else {
		log.Default().Println("Error getting sources: ", err.Error())
	}
	return group, nil
}
