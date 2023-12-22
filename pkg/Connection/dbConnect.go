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
func LoadDB(dbpath string) (bool){

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

	// txn, err := sqliteDB.Begin()
	// if id,err := insertServer(txn, "10.1.0.34", "80"); err != nil {
	// 	fmt.Println("Error while inserting server")
	// 	return nil,true
	// }else{
	// 	fmt.Println("Successfully inserted server ",id)
	// }

	// if id,err := insertServer(txn, "10.1.0.34", "81"); err != nil {
	// 	fmt.Println("Error while inserting server")
	// 	return nil,true
	// } else{
	// 	fmt.Println("Successfully inserted server ",id)
	// }

	// if id,err := insertServer(txn, "127.0.0.1", "8080"); err != nil {
	// 	fmt.Println("Error while inserting server")
	// 	return nil,true
	// }else{
	// 	fmt.Println("Successfully inserted server ",id)
	// }

	if showTable(sqliteDB) != nil {
		fmt.Println("Error while printing table")
		return true
	}

	// if insertPath(txn, 1, "/path2", "1") != nil {
	// 	fmt.Println("Error while inserting path")
	// 	return nil,true
	// }

	// if insertPath(txn, 3, "/path1", "3") != nil {
	// 	fmt.Println("Error while inserting path")
	// 	return nil,true
	// }

	// if insertPath(txn, 2, "/path2", "2") != nil {
	// 	fmt.Println("Error while inserting path")
	// 	return nil,true
	// }

	// if insertClient(txn, 1, "127.0.0.5", "1") != nil {
	// 	fmt.Println("Error while inserting client constraint")
	// 	return nil,true
	// }
	// txn.Commit()
	// localMap := RemoteServer.GenerateMap()
	// remoteServers := make(map[int]*RemoteServer.Server)
	RemoteServer.GenerateMap()
	if loadRemoteServers(sqliteDB, RemoteServer.RemoteServerMap) != nil {
		fmt.Println("Error while loading servers")
		return true
	}
	
	if loadPaths(sqliteDB, RemoteServer.RemoteServerMap) != nil {
		fmt.Println("Error while loading paths")
		return true
	}

	if loadIpConstraint(sqliteDB, RemoteServer.RemoteServerMap) != nil {
		fmt.Println("Error while loading IP constraints")
		return true
	}
	fmt.Println("Remote Server Map : ", RemoteServer.RemoteServerMap)
	return false
}

/*-----------------------------------------------------------------------------------------*/
func HandleDBRequests(dbpath string, serverIP string, serverPort string, paths []string, clients []string) (int,error){
	dbCon, err := sql.Open("sqlite3", dbpath)
	// dbCon, err := sql.Open("sqlite3", dbpath + ":databaselocked.sqlite?cache=shared&mode=rwc")
	var id int
	defer dbCon.Close()

	pathArgs := make([]interface{}, 0)
	clientArgs := make([]interface{}, 0)
	if err != nil {
		return id, err
	}

	txn, err := dbCon.Begin()
	if err != nil {
		return id, err
	}

	id, err = insertServer(txn, serverIP, serverPort)
	if err != nil {
		return id, err
	}

	for _,path := range(paths){
		pathArgs = append(pathArgs, path, fmt.Sprint(id))
	}
	if err := insertPath(txn, id, pathArgs...); err != nil{
		return id, err
	}

	for _,client := range(clients){
		clientArgs = append(clientArgs, client, fmt.Sprint(id))
	}
	if err := insertClient(txn, id, clientArgs...); err != nil{
		return id, err
	}

	err = txn.Commit()
	if err != nil {
		return id, err
	}

	return id, nil

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
			"pathconstraint" boolean DEFAULT false,
			"ipconstraint" boolean DEFAULT false	
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
	txn, err := dbCon.Begin()
	if err != nil {
		return err
	}
	err = QxecuteQuery(txn, fmt.Sprintf(query, tableName))
	if err != nil {
		return err
	}
	return txn.Commit()
}

/*-----------------------------------------------------------------------------------------*/
func loadRemoteServers(dbCon *sql.DB, m *RemoteServer.Map) error{

	query := "SELECT * FROM servers"
	rows, err := dbCon.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() { 
		var id int
		var ipaddress string
		var port string
		var pathConst bool
		var ipConst bool

		rows.Scan(&id, &ipaddress, &port, &pathConst, &ipConst)
		log.Println("Row: ", id, " ", ipaddress, " ", port, " ", pathConst, " ", ipConst)
		
		m.AddServer(id, ipaddress, port, pathConst, ipConst)
	}
	fmt.Println("Servers: ", m)
	return nil
}

/*-----------------------------------------------------------------------------------------*/
func QxecuteQuery(dbCon *sql.Tx, sql string, args ...interface{}) error {

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
func insertServer(dbCon *sql.Tx , ip string, port string) (int,error) {

	var pkid int
	fmt.Println("Inside insert func")
	insertSQL := `INSERT INTO servers(ipaddress, port) VALUES (?, ?) RETURNING pkid`

	statement, err := dbCon.Prepare(insertSQL) 
	if err != nil {
		fmt.Println("error ", err.Error())
		return pkid,err
	}
	err = statement.QueryRow(ip, port).Scan(&pkid)
	if err != nil {

		statement, err = dbCon.Prepare("SELECT pkid FROM servers WHERE ipaddress = ? AND port = ?")
		rows, err := statement.Query(ip, port)
		
		if err != nil {
			return pkid,err
		}
		for rows.Next() { 
			rows.Scan(&pkid)
			log.Println(" Server already exists id: ", pkid)

		}
		rows.Close() 
		return pkid, nil
	}
	log.Println("Inser Query executed successfully created")
	fmt.Println("Inside insert func id  : ", pkid)

	return pkid,nil
}

/*-----------------------------------------------------------------------------------------*/
func insertPath(dbCon *sql.Tx, hostID int, args ...interface{}) error {

	insertSQL := `INSERT INTO pathmappings(path, serverid) VALUES`
	entries := len(args)/2
	for i := 0; i < entries; i++ {
		insertSQL += " (?, ?),"
	}
	err := QxecuteQuery(dbCon, insertSQL[:len(insertSQL)-1], args...)
	updateServer := `UPDATE servers SET pathconstraint = 'TRUE' WHERE pkid =?`
	err = QxecuteQuery(dbCon, updateServer, fmt.Sprint(hostID))
	fmt.Println("Insert path error: ", err)
	return err
}

/*-----------------------------------------------------------------------------------------*/
func insertClient(dbCon *sql.Tx, hostID int, args ...interface{}) error {

	insertSQL := `INSERT INTO addressmappings(ipaddress, serverid) VALUES`
	entries := len(args)/2
	for i := 0; i < entries; i++ {
		insertSQL += " (?, ?),"
	}
	err := QxecuteQuery(dbCon, insertSQL[:len(insertSQL)-1], args...)
	updateServer := `UPDATE servers SET ipconstraint = 'TRUE' WHERE pkid =?`
	err = QxecuteQuery(dbCon, updateServer, fmt.Sprint(hostID))
	fmt.Println("Insert client error: ", err)
	return err
}

/*-----------------------------------------------------------------------------------------*/
func showTable(dbCon *sql.DB) error{

	fmt.Println("*************** SERVER TABLE ****************************")
	showServer := `
		SELECT pkid, ipaddress, port from servers;
		`
	rows, err := dbCon.Query(showServer)
	if err != nil {
		fmt.Println("Error ", err.Error())
		return err
	}
	for rows.Next() {
		var pkid int
		var ipaddress string
		var port string
		rows.Scan(&pkid, &ipaddress, &port)
		fmt.Println("Server ", pkid, " ", ipaddress, " ", port)
	}
	rows.Close() 

	fmt.Println("*************** PATH TABLE ****************************")

	showPath := `
		SELECT pkid, path, serverid from pathmappings;
		`
	rows2, err := dbCon.Query(showPath)	
	if err != nil {
		fmt.Println("Error ", err.Error())
		return err
	}

	for rows2.Next() {
		var pkid int
		var path string
		var serverid int
		rows2.Scan(&pkid, &path, &serverid)
		fmt.Println("Pathid ", pkid, " path ", path, " serverID ", serverid)
	}
	rows2.Close() 

	fmt.Println("*************** Client TABLE ****************************")

	showClient := `
		SELECT pkid, ipaddress, serverid from addressmappings;
		`
	rows3, err := dbCon.Query(showClient)
	if err != nil {
		fmt.Println("Error ", err.Error())
		return err
	}
	for rows3.Next() {
		var pkid int
		var client string
		var serverid int
		rows3.Scan(&pkid, &client, &serverid)
		fmt.Println("ClientID ", pkid, " ClientIP ", client, " serverID ", serverid)
	}
	rows3.Close() 



	return nil
}

/*-----------------------------------------------------------------------------------------*/
func loadPaths(dbCon *sql.DB, m *RemoteServer.Map) error{

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
		// log.Println("server for path: ", servers[id])
		m.UpdatePath(path, id)

	}

	fmt.Println("Paths: ", m)
	return nil

}

/*-----------------------------------------------------------------------------------------*/
func loadIpConstraint(dbCon *sql.DB, m *RemoteServer.Map) error{

	fmt.Println("Inside loadIpConstraint")
	showIpConstraints := `
		SELECT s.pkid, i.ipaddress, s.ipaddress, s.port FROM addressmappings AS i
			LEFT JOIN servers AS s ON s.pkid = i.serverid;
		`
	rows, err := dbCon.Query(showIpConstraints)
	if err != nil {
		fmt.Println("Error ", err.Error())
		return err
	}
	defer rows.Close() 

	for rows.Next() { 
		var clientIp string
		var ipaddress string
		var port string
		var id int
		rows.Scan(&id, &clientIp, &ipaddress, &port)
		log.Println("Row: ", id, " ", clientIp, " ", ipaddress, " ", port)
		// log.Println("server for path: ", servers[id])
		m.UpdateClientIP(clientIp, id)

	}

	fmt.Println("IP map: ", m)
	return nil

}



