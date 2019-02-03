package main

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
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
	tables map[string]map[string]tableInfo
}

// NewDbExplorer creates dbexplorer
func NewDbExplorer(db *sql.DB) (*DbExplorer, error) {
	res := &DbExplorer{
		db:     db,
		tables: make(map[string]map[string]tableInfo),
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
		infos[info.Field] = info
		// infos = append(infos, info)
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
		db.tables[table] = make(map[string]tableInfo)
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
	for k, v := range infos {
		if v.PK {
			return k
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

func (db *DbExplorer) generateInsertUpdateQuery(table string, a interface{}) (columns []string, values []interface{}, err error) {
	m, ok := a.(map[string]interface{})
	if !ok {
		return nil, nil, errors.New("something wrong")
	}

	infos := db.tables[table]
	for _, info := range infos {
		if v, ok := m[info.Field]; ok {
			columns = append(columns, info.Field)
			values = append(values, v)
		}
	}
	return
}

func (db *DbExplorer) createItem(table string, a interface{}) (int64, error) {
	pk := db.getPK(table)
	if pk == "" {
		return 0, nil
	}

	cols, values, err := db.generateInsertUpdateQuery(table, a)
	if err != nil || len(cols) == 0 {
		return 0, err
	}

	// remove PK field from cols
	var index int
	var find bool
	for i, col := range cols {
		index = i
		if col == pk {
			find = true
			break
		}
	}
	if find {
		cols = append(cols[:index], cols[index+1:]...)
		values = append(values[:index], values[index+1:]...)
	}

	// create query
	columns := strings.Join(cols, ", ")
	params := strings.Repeat(", ?", len(cols))
	params = params[1:]
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES(%s)", table, columns, params)
	// fmt.Println(query)

	result, err := db.db.Exec(query, values...)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (db *DbExplorer) updateItem(table, id string, a interface{}) (int64, error) {
	pk := db.getPK(table)
	if pk == "" {
		return 0, nil
	}

	cols, values, err := db.generateInsertUpdateQuery(table, a)
	if err != nil || len(cols) == 0 {
		return 0, err
	}

	//check types
	// for i, col := range cols {

	// }

	columns := strings.Join(cols, " = ?, ")
	columns = columns[:len(columns)-2]

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = ?", table, columns, pk)
	fmt.Println(query)

	values = append(values, id)
	result, err := db.db.Exec(query, values...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
