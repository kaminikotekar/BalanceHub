package Connection

import (
	"database/sql"
	"fmt"
	"os"
	"log"
	"io/ioutil"
	"github.com/kaminikotekar/BalanceHub/pkg/Models/RemoteServer"
	_"github.com/mattn/go-sqlite3" 
)

const(
	SERVER_TABLE_NAME = "servers"
	PATH_MAPPING_TABLE_NAME = "pathmappings"
	ADDRESS_MAPPING_TABLE_NAME = "addressmappings"
)

/*-----------------------------------------------------------------------------------------*/
func LoadDB(dbpath string) bool{

	sqliteDB, err := sql.Open("sqlite3", dbpath)
	defer sqliteDB.Close() 
	if err != nil {
		fmt.Println("Error ,", err)
		fmt.Println("Database, ", sqliteDB)
		return true
	}

	_, error := os.Stat(dbpath)
	if os.IsNotExist(error) {
		fmt.Printf("%v file does not exist\n", dbpath)
		_, err := os.Create(dbpath)  //create a new file
		if err != nil {
			fmt.Println("could not create database")
			return true
		}
		initSQL, err := ioutil.ReadFile("init.sql")
		if err != nil {
			fmt.Println("could not load sql init file")
			return true
		}
		_, err = sqliteDB.Exec(string(initSQL))
		if err != nil{
			fmt.Println("Could not execute init file ", err)
			return true
		}
	}

	if createIfNotExist(sqliteDB) != nil {
		return true
	}
	fmt.Println("Successfully created all tables in database")

	if insertServer(sqliteDB, "10.1.0.34", "80") != nil {
		fmt.Println("Error while inserting server")
		return true
	}

	if insertServer(sqliteDB, "10.1.0.34", "81") != nil {
		fmt.Println("Error while inserting server")
		return true
	}

	if showTable(sqliteDB) != nil {
		fmt.Println("Error while printing table")
		return true
	}

	if insertPath(sqliteDB, "/path1", 1) != nil {
		fmt.Println("Error while inserting path")
		return true
	}

	if insertPath(sqliteDB, "/path2", 2) != nil {
		fmt.Println("Error while inserting path")
		return true
	}

	remoteServers := make(map[int]*RemoteServer.Server)
	if loadRemoteServers(sqliteDB, remoteServers) != nil {
		fmt.Println("Error while loading servers")
		return true
	}

	paths := RemoteServer.GeneratePaths()
	
	if loadPaths(sqliteDB, remoteServers, paths) != nil {
		fmt.Println("Error while loading paths")
		return true
	}

	return false
}

/*-----------------------------------------------------------------------------------------*/
func createIfNotExist(dbCon *sql.DB) error{
	
	var count int
	query := `SELECT count(*) FROM sqlite_master WHERE type='table' AND name='%s'`
	tables  := [3]string{SERVER_TABLE_NAME, PATH_MAPPING_TABLE_NAME, ADDRESS_MAPPING_TABLE_NAME}

	for _,table_name := range tables{
		fmt.Println("table name ", table_name)
		rows, err := dbCon.Query(fmt.Sprintf(query, table_name))
		fmt.Printf("type of row %T", rows)
		fmt.Println("err for create table if exists", err)
		rows.Next()
		rows.Scan(&count)
		rows.Close()
		log.Println("count: ", count)
		if count == 0 {
			err = createTable(dbCon, table_name)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

/*-----------------------------------------------------------------------------------------*/
func createTable(dbCon *sql.DB, tableName string)  error{

	var query string
	if tableName == SERVER_TABLE_NAME {
		query = `
			CREATE TABLE %s (
			"pkid" integer NOT NULL PRIMARY KEY AUTOINCREMENT,		
			"ipaddress" varchar(25),
			"port" varchar(10),
			"constraint" boolean DEFAULT false		
		);`
	} else if tableName == PATH_MAPPING_TABLE_NAME {
		query = `
			CREATE TABLE %s (
			"pkid" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
			"path" text,
			"serverid" integer NOT NULL,
			FOREIGN KEY(serverid) REFERENCES servers(pkid)
		);`
	} else if tableName == ADDRESS_MAPPING_TABLE_NAME{
		query = `
			CREATE TABLE %s (
			"pkid" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
			"ipaddress" varchar(25),
			"serverid" integer NOT NULL,
			FOREIGN KEY(serverid) REFERENCES servers(pkid)
		);`
	}
	
	return QxecuteQuery(dbCon, fmt.Sprintf(query, tableName))
}

/*-----------------------------------------------------------------------------------------*/
func loadRemoteServers(dbCon *sql.DB, servers map[int]*RemoteServer.Server) error{

	query := "SELECT pkid, ipaddress, port FROM servers"
	rows, err := dbCon.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() { 
		var id int
		var ipaddress string
		var port string

		rows.Scan(&id, &ipaddress, &port)
		log.Println("Row: ", id, " ", ipaddress, " ", port)
		
		servers[id] = &RemoteServer.Server{
			Ipaddress: ipaddress,
			Port: port,
		}
	}
	fmt.Println("Servers: ", servers)
	return nil
}

/*-----------------------------------------------------------------------------------------*/
func QxecuteQuery(dbCon *sql.DB, sql string, args ...interface{}) error {

	fmt.Println(" Args in exec query: ", args)
	statement, err := dbCon.Prepare(sql) 
	if err != nil {
		log.Fatal(err.Error())
		fmt.Println("error ", err.Error())
		return err
	}
	_, err = statement.Exec(args...)
	if err != nil {
		return err
	}
	log.Println("Query executed successfully created")
	return nil
}

/*-----------------------------------------------------------------------------------------*/
func insertServer(dbCon *sql.DB , ip string, port string) error {

	fmt.Println("Inside insert func")
	insertSQL := `INSERT INTO servers(ipaddress, port) VALUES (?, ?)`
	err := QxecuteQuery(dbCon, insertSQL, ip, port)
	return err
}

/*-----------------------------------------------------------------------------------------*/
func insertPath(dbCon *sql.DB, path string, hostID int) error {

	insertSQL := `INSERT INTO pathmappings(path, serverid) VALUES (?, ?)`
	err := QxecuteQuery(dbCon, insertSQL, path, fmt.Sprint(hostID))
	return err
}

/*-----------------------------------------------------------------------------------------*/
func showTable(dbCon *sql.DB) error{
	showServer := `
		SELECT pkid, ipaddress, port from servers;
		`
	rows, err := dbCon.Query(showServer)
	if err != nil {
		fmt.Println("Error ", err.Error())
		return err
	}
	defer rows.Close() 
	for rows.Next() {
		var pkid int
		var ipaddress string
		var port string
		rows.Scan(&pkid, &ipaddress, &port)
		fmt.Println("Server ", pkid, " ", ipaddress, " ", port)
	}
	return nil
}

/*-----------------------------------------------------------------------------------------*/
func loadPaths(dbCon *sql.DB, 
				servers map[int]*RemoteServer.Server, 
				paths *RemoteServer.Paths) error{

	fmt.Println("Inside loadPaths")
	showPaths := `
		SELECT s.pkid, p.path, s.ipaddress, s.port FROM pathmappings AS p
			LEFT JOIN servers AS s ON s.pkid = p.serverid;
		`
	rows, err := dbCon.Query(showPaths)
	if err != nil {
		fmt.Println("Error ", err.Error())
		return err
	}
	defer rows.Close() 

	for rows.Next() { 
		var path string
		var ipaddress string
		var port string
		var id int
		rows.Scan(&id, &path, &ipaddress, &port)
		log.Println("Row: ", id, " ", path, " ", ipaddress, " ", port)
		log.Println("server for path: ", servers[id])
		paths.UpdatePath(path, id, servers[id])	
	}

	fmt.Println("Paths: ", paths)
	return nil

}



