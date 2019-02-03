package main

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

var errSomethingWrong = errors.New("something wrong")

const (
	defaultLimit  = 5
	defaultOffset = 0
)

type tableInfo struct {
	Field       string
	Type        string
	ReflectType reflect.Type
	Null        bool
	PK          bool
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

	fields, ok := db.tables[table]
	if !ok {
		return errUnknownTable
	}

	for rows.Next() {
		var collation, defaultValue *string
		var null, key, extra, privs, comment string
		info := tableInfo{}
		if err := rows.Scan(&info.Field, &info.Type, &collation, &null, &key, &defaultValue, &extra, &privs, &comment); err != nil {
			return err
		}
		info.Null = null == "YES"
		info.PK = key == "PRI" && extra == "auto_increment"

		fields[info.Field] = info
	}

	// load reflect type for column
	query = fmt.Sprintf("SELECT * FROM %s LIMIT 1", table)
	rows, err = db.db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	// cols, err := rows.ColumnTypes()
	// if err != nil {
	// 	return err
	// }

	// for _, col := range cols {
	// 	info, ok := fields[col.Name()]
	// 	if !ok {
	// 		continue
	// 	}
	// 	info.ReflectType = col.ScanType()
	// 	fields[col.Name()] = info
	// }

	// fmt.Println(fields)

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
	fields := db.tables[table]
	for k, info := range fields {
		if info.PK {
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
		return nil, nil, errSomethingWrong
	}

	fields := db.tables[table]
	for field := range fields {
		if v, ok := m[field]; ok {
			columns = append(columns, field)
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
	if len(cols) > 1 {
		params = params[1:]
	}
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES(%s)", table, columns, params)
	// fmt.Println(query)

	result, err := db.db.Exec(query, values...)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (db *DbExplorer) updateItem(table, id string, a interface{}) (int64, error) {
	m, ok := a.(map[string]interface{})
	if !ok {
		return 0, errSomethingWrong
	}

	pk := db.getPK(table)
	if pk == "" {
		return 0, nil
	}

	cols, values, err := db.generateInsertUpdateQuery(table, a)
	if err != nil || len(cols) == 0 {
		return 0, err
	}

	// check types
	fields := db.tables[table]
	for _, col := range cols {
		field, ok := fields[col]
		err := fmt.Errorf("field %s have invalid type", col)

		if !ok {
			continue
		}

		if field.PK {
			return 0, err
		}

		switch t := m[col].(type) {
		case int:
			if !strings.Contains(field.Type, "int") {
				return 0, err
			}
		case string:
			if !strings.Contains(field.Type, "varchar") && !strings.Contains(field.Type, "text") {
				return 0, err
			}
		case float32, float64:
			if !strings.Contains(field.Type, "float") && !strings.Contains(field.Type, "double") {
				return 0, err
			}
		case nil:
			if !field.Null {
				return 0, err
			}
		default:
			fmt.Printf("type: %s %T\n", field.Type, t)
		}
	}

	columns := strings.Join(cols, " = ?, ")
	columns += " = ? "

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = ?", table, columns, pk)
	values = append(values, id)
	result, err := db.db.Exec(query, values...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
