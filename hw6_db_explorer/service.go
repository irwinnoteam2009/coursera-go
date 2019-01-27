package main

import (
	"database/sql"
	"fmt"
)

const (
	defaultLimit  = 5
	defaultOffset = 0
)

// DbExplorer ...
type DbExplorer struct {
	db *sql.DB
}

// NewDbExplorer creates dbexplorer
func NewDbExplorer(db *sql.DB) (*DbExplorer, error) {
	return &DbExplorer{db: db}, nil
}

func (db *DbExplorer) tableExists(table string) error {
	tables, err := db.getTableList()
	if err != nil {
		return err
	}
	for _, t := range tables {
		if t == table {
			return nil
		}
	}
	return errUnknownTable
}

func (db *DbExplorer) getPK(table string) string {
	query := fmt.Sprintf("SHOW KEYS FROM %s WHERE Key_name = 'PRIMARY';", table)
	rows, err := db.db.Query(query)
	if err != nil {
		return ""
	}
	defer rows.Close()

	for rows.Next() {
		item, err := scanStruct(rows)
		if err != nil {
			return ""
		}
		m := item.(map[string]interface{})
		if v, ok := m["Column_name"]; ok {
			col := v.(string)
			return col
		}

	}
	return ""
}

func (db *DbExplorer) getTableList() ([]string, error) {
	rows, err := db.db.Query("SHOW TABLES;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tables := make([]string, 0)
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	return tables, nil
}

func (db *DbExplorer) getItemsList(table string, limit, offset int) ([]interface{}, error) {
	if err := db.tableExists(table); err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT * FROM %s LIMIT ?, ?", table)
	if limit == 0 {
		limit = defaultLimit
	}
	rows, err := db.db.Query(query, offset, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]interface{}, 0)
	for rows.Next() {
		item, err := scanStruct(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	fmt.Println("items:", items)
	fmt.Println("PK:", db.getPK(table))
	return items, nil
}

func (db *DbExplorer) itemExists(table string, id interface{}) error {
	pk := db.getPK(table)
	if pk == "" {
		return errRecordNotFound
	}

	var count int
	query := fmt.Sprintf("SELECT COUNT(1) FROM %s WHERE %s = ?", table, pk)
	row := db.db.QueryRow(query, id)
	return row.Scan(&count)
}

func (db *DbExplorer) getItem(table, id string) (interface{}, error) {
	if err := db.tableExists(table); err != nil {
		return nil, err
	}
	// if err := db.itemExists(table, id); err != nil {
	// 	fmt.Println("getItem:", err)
	// 	return nil, err
	// }
	pk := db.getPK(table)
	if pk == "" {
		return nil, errRecordNotFound
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE %s = ?", table, pk)
	rows, err := db.db.Query(query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		item, err := scanStruct(rows)
		if err != nil {
			return nil, err
		}
		return item, nil
	}

	return nil, errRecordNotFound
}

func (db *DbExplorer) deleteItem(table, id string) (int64, error) {
	if err := db.tableExists(table); err != nil {
		return 0, err
	}
	pk := db.getPK(table)
	if pk == "" {
		return 0, nil
	}
	query := fmt.Sprintf("DELETE FROM %s WHERE %s = ?", table, pk)
	result, err := db.db.Exec(query, id)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (db *DbExplorer) createItem(table string, a interface{}) (int64, error) {
	if err := db.tableExists(table); err != nil {
		return 0, err
	}

	return 0, nil
}

func (db *DbExplorer) updateItem(table string, a interface{}) (int64, error) {
	if err := db.tableExists(table); err != nil {
		return 0, err
	}
	return 0, nil
}
