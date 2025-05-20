package db

import (
	"database/sql"
	"time"
)

type List struct {
	ID          uint64    `db:"id" json:"id"`
	Time        time.Time `db:"time" json:"date"`
	Title       string    `db:"title" json:"title"`
	Descritpion string    `db:"description" json:"description,omitempty"`
	Enclosure   *string   `db:"enclosure" json:"enclosure,omitempty"`
	IsRT        bool      `db:"is_rt" json:"isRT"`
	SourceName  string    `db:"source_name" json:"sourceName"`
}

type Source struct {
	Title       string         `json:"title" db:"source_title"`                       // Заголовок источника
	SourceName  string         `json:"name" db:"source_name"`                         // Имя источника (например, "BBC News")
	Time        time.Time      `json:"pubDate" db:"source_time"`                      // Время публикации источника
	Link        string         `json:"link" db:"source_link"`                         // Ссылка на оригинальный источник
	Description sql.NullString `json:"description,omitempty" db:"source_description"` // Описание из источника
	FullText    sql.NullString `json:"full_text" db:"source_full_text"`               // Полный текст из источника
	Enclosure   sql.NullString `json:"enclosure,omitempty" db:"source_enclosure"`     // Обложка из источника
}

type News struct {
	ID          uint64         `json:"id" db:"id"`
	Title       string         `json:"title" db:"title"`                       // Это теперь заголовок ГРУППЫ
	Description sql.NullString `json:"description,omitempty" db:"description"` // Описание ГРУППЫ
	Time        time.Time      `json:"date" db:"time"`                         // Время создания ГРУППЫ
	FullText    sql.NullString `json:"rewrite" db:"full_text"`                 // Полный текст ГРУППЫ (вероятно, rewrite)
	Enclosure   sql.NullString `json:"enclosure,omitempty" db:"enclosure"`     // Обложка ГРУППЫ
	Sources     []Source       `json:"sources" db:"-"`                         // Массив дочерних источников
	ViewsCount  uint64         `json:"viewsCount" db:"views_count"`            // Счетчик просмотров группы
}
