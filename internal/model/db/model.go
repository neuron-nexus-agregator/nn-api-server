package db

import "time"

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
	Title       string    `db:"title" json:"title"`
	SourceName  string    `db:"source_name" json:"name"`
	Time        time.Time `db:"time" json:"pubDate"`
	Link        string    `db:"link" json:"link"`
	Descritpion string    `db:"description" json:"description,omitempty"`
	FullText    string    `db:"full_text" json:"fullText"`
	Enclosure   *string   `db:"enclosure" json:"enclosure,omitempty"`
}

type News struct {
	ID          uint64    `db:"id" json:"id"`
	Title       string    `db:"title" json:"title"`
	Descritpion string    `db:"description" json:"description,omitempty"`
	Time        time.Time `db:"time" json:"date"`
	FullText    string    `db:"full_text" json:"rewrite"`
	Enclosure   *string   `db:"enclosure" json:"enclosure,omitempty"`
	Sources     []Source  `db:"-" json:"sources"`
}
