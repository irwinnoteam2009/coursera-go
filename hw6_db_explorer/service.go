package main

import "database/sql"

// DbExplorer ...
type DbExplorer struct {
	db *sql.DB
}

// NewDbExplorer creates dbexplorer
func NewDbExplorer(db *sql.DB) (*DbExplorer, error) {
	return &DbExplorer{db: db}, nil
}

func (db *DbExplorer) isTableExists(table string) bool {
	return true
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
		err := rows.Scan(&table)
		if err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	return tables, nil
}

func (db *DbExplorer) getItemsList(table string, limit, offset int) ([]interface{}, error) {
	if !db.isTableExists(table) {
		return nil, errUnknownTable
	}
	return nil, nil
}

func (db *DbExplorer) getItem(table, id string) (interface{}, error) {
	if !db.isTableExists(table) {
		return nil, errUnknownTable
	}
	return nil, nil
}

func (db *DbExplorer) deleteItem(table, id string) (int, error) {
	if !db.isTableExists(table) {
		return 0, errUnknownTable
	}
	// affected count
	return 0, nil
}

func (db *DbExplorer) createItem(table string, a interface{}) (int, error) {
	if !db.isTableExists(table) {
		return 0, errUnknownTable
	}
	return 0, nil
}

func (db *DbExplorer) updateItem(table string, a interface{}) (int, error) {
	if !db.isTableExists(table) {
		return 0, errUnknownTable
	}
	return 0, nil
}
