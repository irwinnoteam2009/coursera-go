package main

import (
	"database/sql"
	"errors"
	"fmt"
)

const (
	defaultLimit  = 5
	defaultOffset = 0
)

type tableInfo struct {
	Field string
	Type  string
	Null  bool
	PK    bool
}

// DbExplorer ...
type DbExplorer struct {
	db     *sql.DB
	tables map[string][]tableInfo
}

// NewDbExplorer creates dbexplorer
func NewDbExplorer(db *sql.DB) (*DbExplorer, error) {
	res := &DbExplorer{
		db:     db,
		tables: make(map[string][]tableInfo),
	}

	if err := res.loadTables(); err != nil {
		return nil, err
	}
	return res, nil
}

func (db *DbExplorer) loadTableInfo(table string) error {
	query := fmt.Sprintf("SHOW FULL COLUMNS FROM %s;", table)
	rows, err := db.db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var collation, defaultValue *string
		var null, key, extra, privs, comment string
		info := tableInfo{}
		if err := rows.Scan(&info.Field, &info.Type, &collation, &null, &key, &defaultValue, &extra, &privs, &comment); err != nil {
			return err
		}
		info.Null = null == "YES"
		info.PK = key == "PRI" && extra == "auto_increment"

		infos, ok := db.tables[table]
		if !ok {
			return errUnknownTable
		}
		infos = append(infos, info)
		db.tables[table] = infos
	}

	return nil
}

func (db *DbExplorer) loadTables() error {
	rows, err := db.db.Query("SHOW TABLES;")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return err
		}
		db.tables[table] = make([]tableInfo, 0)
	}

	for table := range db.tables {
		if err := db.loadTableInfo(table); err != nil {
			return err
		}
	}

	return nil
}

func (db *DbExplorer) tableExists(table string) error {
	if _, ok := db.tables[table]; !ok {
		return errUnknownTable
	}
	return nil
}

func (db *DbExplorer) getPK(table string) string {
	infos := db.tables[table]
	for _, info := range infos {
		if info.PK {
			return info.Field
		}
	}
	return ""
}

func (db *DbExplorer) getTableList() ([]string, error) {
	res := make([]string, 0, len(db.tables))
	for table := range db.tables {
		res = append(res, table)
	}
	return res, nil
}

func (db *DbExplorer) getItemsList(table string, limit, offset int) ([]interface{}, error) {
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
	// get ctable column list
	rows, err := db.db.Query("SELECT * FROM " + table + " LIMIT 1")
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return 0, err
	}

	m, ok := a.(map[string]interface{})
	if !ok {
		return 0, errors.New("something wrong")
	}
	// create column set, params and query values
	values := make([]interface{}, 0)
	var columns, params string
	for _, c := range cols {
		if v, ok := m[c]; ok {
			columns += ", " + c
			params += ", ?"
			values = append(values, v)
		}
	}
	if len(columns) != 0 {
		columns = columns[1:]
		params = params[1:]
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES(%s)", table, columns, params)
	fmt.Println(query)

	result, err := db.db.Exec(query, values...)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (db *DbExplorer) updateItem(table string, a interface{}) (int64, error) {
	return 0, nil
}
