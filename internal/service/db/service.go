package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"agregator/api/internal/interfaces"
	model "agregator/api/internal/model/db"
)

type DB struct {
	db     *sqlx.DB
	logger interfaces.Logger
}

type newsDB struct {
	ID          uint64          `db:"id"`
	Title       string          `db:"title"`
	Description sql.NullString  `db:"description"`
	FullText    sql.NullString  `db:"full_text"`
	Time        time.Time       `db:"time"`
	Enclosure   sql.NullString  `db:"enclosure"`
	ViewsCount  uint64          `db:"views_count"`
	SourcesJSON json.RawMessage `db:"sources_json"` // Здесь будет JSON-массив источников
}

func New(logger interfaces.Logger) (*DB, error) {

	connectionData := fmt.Sprintf("user=%s dbname=%s sslmode=disable password=%s host=%s port=%s", os.Getenv("DB_LOGIN"), "newagregator", os.Getenv("DB_PASSWORD"), os.Getenv("DB_HOST"), os.Getenv("DB_PORT"))
	db, err := sqlx.Connect("postgres", connectionData)

	return &DB{
		db:     db,
		logger: logger,
	}, err
}

func (g *DB) GetLastIndex() (uint64, error) {
	var index uint64
	err := g.db.QueryRow("SELECT MAX(id) FROM groups").Scan(&index)
	if err != nil {
		g.logger.Error("Error getting last index", "error", err)
		return 0, err
	}
	return index, nil
}

func (g *DB) Get(lastDate time.Time, limit uint64, search ...string) ([]model.List, error) {
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
	stmt, err := g.db.Preparex(baseReq)
	if err != nil {
		g.logger.Error("Error preparing statement", "error", err.Error())
		return nil, err
	}
	defer stmt.Close()

	var groups []model.List
	err = stmt.Select(&groups, args...)
	if err != nil {
		g.logger.Error("Error executing statement", "error", err.Error())
		return nil, err
	}

	return groups, nil
}

func (g *DB) GetTopGroupsByFeedCount(limit uint64) ([]model.List, error) {
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

	stmt, err := g.db.Preparex(req)
	if err != nil {
		g.logger.Error("Error preparing query", "error", err.Error())
		return nil, err
	}
	defer stmt.Close()

	var groups []model.List
	err = stmt.Select(&groups, limit)
	if err != nil {
		g.logger.Error("Error executing query", "error", err.Error())
		return nil, err
	}

	return groups, nil
}

func (g *DB) GetRTGroups(limit uint64, is_rt bool) ([]model.List, error) {
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

	stmt, err := g.db.Preparex(req)
	if err != nil {
		g.logger.Error("Error preparing query", "error", err.Error())
		return nil, err
	}
	defer stmt.Close()

	var groups []model.List
	err = stmt.Select(&groups, is_rt, limit)
	if err != nil {
		g.logger.Error("Error executing query", "error", err.Error())
		return nil, err
	}

	return groups, nil
}

func (g *DB) GetSimilarGroups(id, limit uint64) ([]model.List, error) {
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
	err := g.db.Select(&groups, req, id, limit)
	if err != nil {
		g.logger.Error("Error executing query", "error", err.Error())
		return nil, err
	}

	return groups, nil
}

// GetByID теперь получает группу и все ее источники за один запрос
func (g *DB) GetByID(id uint64) (model.News, error) {
	req := `
    SELECT
        g.id,
        fc.title,
        fc.description,
        fc.full_text,
        g.time,
        g.views,
        (
            SELECT COALESCE(f_enc.enclosure, '')
            FROM compares AS c_enc
            JOIN feed AS f_enc ON f_enc.id = c_enc.feed_id
            WHERE c_enc.group_id = g.id
              AND f_enc.enclosure IS NOT NULL
              AND f_enc.enclosure != ''
            LIMIT 1
        ) AS enclosure,
        COALESCE(
            json_agg(
                json_build_object(
                    'title', fc.title,
                    'link', fc.link,
                    'name', fc.source_name,
                    'pubDate', fc.time,
                    'description', fc.description,
                    'fullText', fc.full_text,
                    'enclosure', fc.enclosure
                ) ORDER BY fc.time DESC, fc.id
            ) FILTER (WHERE fc.id IS NOT NULL),
        '[]'::json) AS sources_json
    FROM
        groups AS g
    LEFT JOIN
        compares AS cp ON cp.group_id = g.id
    LEFT JOIN
        feed AS fc ON fc.id = cp.feed_id
    WHERE
        g.id = $1
    GROUP BY
        g.id, g.title, g.description, g.full_text, g.time, g.views`

	var dbNews newsDB
	err := g.db.Get(&dbNews, req, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.News{}, fmt.Errorf("group with ID %d not found: %w", id, err)
		}
		g.logger.Error("Error executing query", "error", err.Error())
		return model.News{}, fmt.Errorf("failed to query group %d: %w", id, err)
	}

	var sources []model.Source
	// Демаршалируем JSON-массив источников
	if len(dbNews.SourcesJSON) > 0 {
		err = json.Unmarshal(dbNews.SourcesJSON, &sources)
		if err != nil {
			g.logger.Error("Error parsing sources for group", "error", err.Error(), "id", id)
			// В случае ошибки демаршалинга, можно вернуть ошибку или пустой слайс sources
			// В данном случае, возвращаем ошибку, так как это может быть критично.
			return model.News{}, fmt.Errorf("failed to parse sources for group %d: %w", id, err)
		}
	}

	// Собираем конечную структуру model.News
	group := model.News{
		ID:          dbNews.ID,
		Title:       dbNews.Title,
		Description: dbNews.Description, // Уже sql.NullString
		FullText:    dbNews.FullText,    // Уже sql.NullString
		Time:        dbNews.Time,
		Enclosure:   dbNews.Enclosure, // Уже sql.NullString
		ViewsCount:  dbNews.ViewsCount,
		Sources:     sources,
	}

	return group, nil
}

func (g *DB) IncrementVies(id uint64) error {
	req := `UPDATE groups SET views = views + 1 WHERE id = $1`
	_, err := g.db.Exec(req, id)
	if err != nil {
		g.logger.Error("Error executing query", "error", err.Error())
		return err
	}
	return nil
}

func (g *DB) UpdateViews(id uint64, views uint64) error {
	req := `UPDATE groups SET views = views + $1 WHERE id = $2`
	_, err := g.db.Exec(req, views, id)
	if err != nil {
		g.logger.Error("Error executing query", "error", err.Error())
		return err
	}
	return nil
}

func (g *DB) UpdateViewsBatch(views map[int64]int64) error {
	tx, err := g.db.Begin()
	if err != nil {
		g.logger.Error("Error starting transaction", "error", err.Error())
		return err
	}
	for id, views := range views {
		_, err := tx.Exec(`UPDATE groups SET views = views + $1 WHERE id = $2`, views, id)
		if err != nil {
			g.logger.Error("Error updating views", "error", err.Error())
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		g.logger.Error("Error committing transaction", "error", err.Error())
		return err
	}
	return nil
}
